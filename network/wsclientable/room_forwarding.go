package wsclientable

import (
	"encoding/json"
	"log"
	"net/url"
)

//Idea: Rooms are additional fields in the ws upgrade request header.
//      The server validates that the room is registered and will inform clients of room constraints.
//         The server will disconnect from clients in rooms that become invalid (timeout or removed by controller)
//         The server cannot force clients to disconnect in a p2p connection, but this is meant to ask them to.
//      Rooms can have allowed clients
//         Specified in the config, if not a single client is specified as allowed ANY client will be allowed.
//      Rooms can be permanent (created through config file)
//      Rooms can be temporary (created through local(!) http requests)
//         Security here relies on the fact that the controller port is not forwarded.
//         Temporary rooms have a date-timeframe
//         Temporary rooms can be deleted

type Room struct {
	ID               string
	allowedClients   map[string]bool // always true, just used because somehow go does not support search in slice
	connectedClients map[string]*ClientConnection
}

type RoomI interface {
	GetID() string
	IsAllowed(userID string) bool

	GetConnectedByID(userID string) *ClientConnection
	AddConnectedIfNotConnected(ClientConnection) bool
	RemoveConnected(ClientConnection)
	Close() error
}

func (r Room) GetConnectedByID(userID string) *ClientConnection {
	if conn, exists := r.connectedClients[userID]; exists {
		return conn
	}

	return nil
}
func (r Room) AddConnectedIfNotConnected(client ClientConnection) bool {
	_, userID := clientConnectionIDStringToRoomIDAndUserID(client.ID)
	// not thread safe:
	_, exists := r.connectedClients[userID]
	if !exists {
		r.connectedClients[userID] = &client
		return true
	}
	return false
}
func (r Room) RemoveConnected(client ClientConnection) {
	_, userID := clientConnectionIDStringToRoomIDAndUserID(client.ID)
	delete(r.connectedClients, userID)
}
func (r Room) GetID() string {
	return r.ID
}
func (r Room) IsAllowed(userID string) bool {
	if len(r.allowedClients) == 0 {
		return true
	}

	_, ok := r.allowedClients[userID]
	return ok
}
func (r Room) Close() error {
	var err error

	for _, conn := range r.connectedClients {
		e := conn.Close()
		if e != nil {
			err = e
		}
	}

	return err
}

type RoomController interface {
	init()
	exists(roomID string) bool
	get(roomID string) RoomI
	close() error
}

type RoomControllers struct {
	controllers []RoomController
}

func BundleControllers(controllers ...RoomController) RoomControllers {
	return RoomControllers{controllers}
}

func (r *RoomControllers) init() {
	for _, v := range r.controllers {
		v.init()
	}
}
func (r *RoomControllers) exists(roomID string) bool {
	for _, v := range r.controllers {
		if v.exists(roomID) {
			return true
		}
	}

	return false
}
func (r *RoomControllers) get(roomID string) RoomI {
	for _, v := range r.controllers {
		r := v.get(roomID)
		if r != nil {
			return r
		}
	}

	return nil
}
func (r *RoomControllers) close() error {
	var err error
	for _, v := range r.controllers {
		e := v.close()
		if e != nil {
			err = e
		}
	}
	return err
}

// Will add direct relay functionality within rooms (described above)
func (s *Server) AddRoomForwardingFunctionality(roomControllers RoomControllers, messageTypes ...string) {
	rooms := roomControllers
	rooms.init()

	roomAuthenticator := func(initialParams url.Values) (string, error) {
		roomID := initialParams.Get("room")
		if len(roomID) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "room"}
		}

		room := rooms.get(roomID)
		if room == nil {
			return "", AuthenticationError{Reason: "Could not find room: " + roomID}
		}

		userID := initialParams.Get("user")
		if len(userID) == 0 {
			return "", MissingURLFieldError{MissingFieldName: "user"}
		}

		if !room.IsAllowed(userID) {
			return "", AuthenticationError{Reason: "User(" + userID + ") not allowed in room: " + roomID}
		}
		convertedClientID := roomIDAndUserIDToClientConnectionIDString(roomID, userID)
		return convertedClientID, nil
	}

	s.SetAuthenticator(roomAuthenticator)

	s.AddServerClosedHandler(func() {
		_ = rooms.close()
	})

	s.AddConnOpenedHandler(func(client ClientConnection) {
		roomID, userID := clientConnectionIDStringToRoomIDAndUserID(client.ID)
		log.Print("Connect: " + userID + ", in " + roomID)

		if room := rooms.get(roomID); room == nil {
			log.Print("Can no longer find room " + roomID)
		} else if !room.AddConnectedIfNotConnected(client) { // this is not thread safe
			_ = client.Close()
		}
	})
	s.AddConnClosedHandler(func(client ClientConnection, code int, reason string) {
		roomID, userID := clientConnectionIDStringToRoomIDAndUserID(client.ID)
		log.Print("Disconnect: " + userID + ", in " + roomID)

		if room := rooms.get(roomID); room == nil {
			log.Print("Can no longer find room " + roomID)
		} else {
			room.RemoveConnected(client)
		}
	})

	directRelayWithinRoom := func(mType string, client ClientConnection, data map[string]interface{}) {
		if to, ok := data["to"]; !ok || to == nil {
			log.Printf("Missing field for message of type %v", mType)
			return
		}

		roomID, userID := clientConnectionIDStringToRoomIDAndUserID(client.ID)

		room := rooms.get(roomID)
		if room == nil {
			log.Print("Client's(" + userID + ") room(" + roomID + ") no longer exists - closing client connection")
			log.Print("This should never happen." +
				"If a room is closed, all clients should be disconnected and no new clients accepted." +
				"Getting a request here is either an unlikely race condition or a bug. Or both. Anyway.")
			_ = client.Close()
			return
		}

		to := data["to"].(string)
		data["from"] = userID
		peer := room.GetConnectedByID(to)
		if peer == nil {
			err := client.SendTyped("error",
				"{\"requestType\":\""+mType+"\", \"reason\":\"Peer "+to+" not found in room "+roomID+"\"}")
			if err != nil {
				log.Printf("Error sending to %v from %v in room %v", to, userID, roomID)
			}
			return
		}

		// relay to other client
		err := peer.SendMapTyped(mType, data)
		if err != nil {
			log.Printf("Error sending to %v from %v in room %v", to, userID, roomID)
		}
	}

	for _, mType := range messageTypes {
		s.AddMessageHandler(mType, directRelayWithinRoom)
	}
}

// yes, I know the following is ugly, but golang is seriously missing support for generics and this is kinda mostly ok.
func clientConnectionIDStringToRoomIDAndUserID(clientID string) (string, string) {
	var clientIDJSON map[string]string
	if err := json.Unmarshal([]byte(clientID), &clientIDJSON); err != nil {
		log.Panic("cannot unmarshal - should never occur, err: " + err.Error())
	}
	return clientIDJSON["r"], clientIDJSON["u"]
}
func roomIDAndUserIDToClientConnectionIDString(roomID, userID string) string {
	return "{\"r\":\"" + roomID + "\", \"u\":\"" + userID + "\"}"
}

func createAllowedIdsMapFromJSONArray(allowedClientsJSONArray string) map[string]bool {
	var allowedClientIds []string
	err := json.Unmarshal([]byte(allowedClientsJSONArray), &allowedClientIds)
	if err != nil {
		log.Panic("Could not load permanent room allowed Ids (not a list: " + allowedClientsJSONArray + ")")
	}
	return createAllowedIdsMapFromSlice(allowedClientIds)
}

func createAllowedIdsMapFromSlice(allowedClientIds []string) map[string]bool {
	allowedClientIdsMap := make(map[string]bool)
	for i := 0; i < len(allowedClientIds); i++ {
		allowedClientIdsMap[allowedClientIds[i]] = true
	}
	return allowedClientIdsMap
}
