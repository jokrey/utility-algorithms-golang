package wsclientable

import (
	"encoding/json"
	"log"
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

// Will add direct relay functionality within rooms (described above)
func (s *Server) AddRoomForwardingFunctionality(roomControllers RoomControllers, messageTypes ...string) {
	rooms := roomControllers
	rooms.Init()

	s.SetAuthenticator(AuthenticateRoomUserPermitAllowed(&rooms))

	s.AddServerClosedHandler(func() {
		_ = rooms.Close()
	})

	s.AddConnOpenedHandler(func(connection ClientConnection) {
		roomID, userID, e := ConnectionIDStringToRoomIDAndUserID(connection.ID)
		if e != nil {
			log.Printf("New Connection, but invalid connection ID(%v), closing", connection.ID)
			_ = connection.Close() //when we could not add the connection to any room, we close it
			return
		}
		log.Print("Connect: " + userID + ", in " + roomID)
		if !rooms.NewConnectionForRoom(roomID, connection) {
			_ = connection.Close() //when we could not add the connection to any room, we close it
		}
	})
	s.AddConnClosedHandler(func(connectionID string, code int, reason string) {
		roomID, userID, e := ConnectionIDStringToRoomIDAndUserID(connectionID)
		if e != nil {
			log.Println("Disconnect:", "Invalid ConnectionID(", connectionID, ")", " :::: RAW(can look strange, might be normal): c=", code, ", r=", reason, ")")
		} else {
			log.Println("Disconnect:", userID, ", in", roomID, " :::: RAW(can look strange, might be normal): c=", code, ", r=", reason, ")")
		}
		rooms.ConnectionInRoomClosed(roomID, userID)
	})

	directRelayWithinRoom := func(mType string, client ClientConnection, data map[string]interface{}) {
		if to, ok := data["to"]; !ok || to == nil {
			log.Printf("Missing field for message of type %v", mType)
			return
		}

		roomID, userID, _ := ConnectionIDStringToRoomIDAndUserID(client.ID)

		room := rooms.GetRoom(roomID)
		if room == nil {
			log.Print("Client's(" + userID + ") room(" + roomID + ") no longer exists - closing client connection")
			log.Print("This should never happen." +
				"If a room is closed, all clients should be disconnected and no new clients accepted." +
				"Getting a request here is either an unlikely race condition or a bug. Or both. Anyway, closing now.")
			_ = client.Close()
			return
		}

		to := data["to"].(string)
		data["from"] = userID

		peer := rooms.GetConnectionInRoom(roomID, to)
		//log.Printf("Attempt send from(%v), to(%v), peer(%v), in room(%v)", userID, to, peer, roomID)
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

func UnmarshalJsonArray(jsonArray string) []string {
	var strs []string
	err := json.Unmarshal([]byte(jsonArray), &strs)
	if err != nil {
		log.Panic("Could not load permanent room allowed Ids (not a list: " + jsonArray + ")")
	}
	return strs
}

func CreateAllowedIdsMapFromSlice(allowedClientIds []string) map[string]bool {
	allowedClientIdsMap := make(map[string]bool)
	for i := 0; i < len(allowedClientIds); i++ {
		allowedClientIdsMap[allowedClientIds[i]] = true
	}
	return allowedClientIdsMap
}
