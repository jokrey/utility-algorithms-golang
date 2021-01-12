package wsclientable

import (
	"gopkg.in/ini.v1"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//Idea: See 'room_forwarding_http_editable.go'
//      Additionally we have two more fields in the http request headers:
//     		'valid_from_in_seconds_from_now' and 'valid_until_in_seconds_from_now'
//          where 'valid_until_in_seconds_from_now' > 'valid_from_in_seconds_from_now' must hold
//          we use the 'seconds from now' semantic to minimize the problem of possibly differences in system time
//          	(it will be properly converted and stored in unix-time timestamps)
//              (note: since we assume localhost, this is likely not required - though nice to have)
//      A temporary room will only show as 'existing' for the given time frame.
//          e.g. clients can only connect within the given time and will be disconnected at the end of that timeframe
//  example editing requests (python3) - note: the concrete required call depends on the config file:
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/add?id=test&allowed_clients=["c", "s", "parent"]&valid_from_in_seconds_from_now=10&valid_until_in_seconds_from_now=1000"); print(r.reason, r.text)
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/edit?id=test&allowed_clients=["c", "s", "parent", "admin"]&valid_from_in_seconds_from_now=0&valid_until_in_seconds_from_now=20"); print(r.reason, r.text)
//         NOTE: when editing the seconds_from_now is calculated again,
//                so it makes sense to have valid_from_in_seconds_from_now=0,
//                  otherwise some might be temporarily in an invalid room (leads to automatic disconnect)
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/remove?id=test"); print(r.reason, r.text)

type TemporaryRoom struct {
	Room
	validFromUnixTime  int64
	validUntilUnixTime int64
}

type TemporaryRoomController struct {
	rooms           map[string]TemporaryRoom
	raw             *http.Server
	bindAddress     string
	bindPort        int
	addRoomRoute    string
	editRoomRoute   string
	removeRoomRoute string
}

func NewTemporaryRoomControllerFromCFG(cfg *ini.File) *TemporaryRoomController {
	bindAddress := cfg.Section("http_temporary_room_controller").Key("address").String()
	bindPort, _ := cfg.Section("http_temporary_room_controller").Key("port").Int()
	addRoomRoute := cfg.Section("http_temporary_room_controller").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_temporary_room_controller").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_temporary_room_controller").Key("remove_room_route").String()
	return NewTemporaryRoomController(bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute)
}

func NewTemporaryRoomController(
	bindAddress string,
	bindPort int,
	addRoomRoute, editRoomRoute, removeRoomRoute string,
) *TemporaryRoomController {
	return &TemporaryRoomController{
		rooms:           make(map[string]TemporaryRoom),
		raw:             nil,
		bindAddress:     bindAddress,
		bindPort:        bindPort,
		addRoomRoute:    addRoomRoute,
		editRoomRoute:   editRoomRoute,
		removeRoomRoute: removeRoomRoute,
	}
}

// implement interface RoomController:

func (p *TemporaryRoomController) exists(roomID string) bool {
	room, exists := p.rooms[roomID]
	if !exists {
		return false
	}
	currentUnixTime := time.Now().Unix()
	return room.validFromUnixTime <= currentUnixTime && currentUnixTime <= room.validUntilUnixTime
}
func (p *TemporaryRoomController) get(roomID string) RoomI {
	if !p.exists(roomID) { // simpler, because the complex logic does not need to be replicated...
		return nil
	}
	v, exists := p.rooms[roomID]
	if exists {
		return v
	}
	return nil
}
func (p *TemporaryRoomController) close() error {
	for _, v := range p.rooms {
		_ = v.Close()
	}
	return p.raw.Close()
}

func (p *TemporaryRoomController) init() {
	go func() {
		handler := http.NewServeMux() // required for concurrent server creation used in tests...

		addOrEditRouteFunc := func(writer http.ResponseWriter, request *http.Request) {
			initialParams, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("could not parse url"))
				return
			}
			isAddRoute := strings.HasPrefix(request.RequestURI, p.addRoomRoute) || strings.HasPrefix(request.RequestURI, "/"+p.addRoomRoute)

			roomID := initialParams.Get("id")
			allowedClientIdsRaw := initialParams.Get("allowed_clients")
			validFromInSecondsFromNowRaw := initialParams.Get("valid_from_in_seconds_from_now")
			validUntilInSecondsFromNowRaw := initialParams.Get("valid_until_in_seconds_from_now")

			validFromInSecondsFromNow, errF := strconv.ParseInt(validFromInSecondsFromNowRaw, 10, 64)
			validUntilInSecondsFromNow, errU := strconv.ParseInt(validUntilInSecondsFromNowRaw, 10, 64)

			if len(roomID) == 0 {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("missing field in url params(string): id"))
				return
			}
			if len(allowedClientIdsRaw) == 0 {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("missing field in url params(json array string): allowed_clients"))
				return
			}
			if errF != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("field in url params(missing or not a number): valid_from_in_seconds_from_now"))
				return
			}
			if errU != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("field in url params(missing or not a number): valid_until_in_seconds_from_now"))
				return
			}

			allowedClientIdsMap := createAllowedIdsMapFromJSONArray(allowedClientIdsRaw)
			currentUnixTime := time.Now().Unix()

			var connectedClients map[string]*ClientConnection
			if previousRoom, exists := p.rooms[roomID]; exists {
				if isAddRoute {
					writer.WriteHeader(http.StatusForbidden)
					_, _ = writer.Write([]byte("room exists, use edit route to edit"))
					return
				}
				connectedClients = previousRoom.connectedClients
			} else {
				connectedClients = make(map[string]*ClientConnection)
			}
			newRoom := TemporaryRoom{
				Room: Room{
					ID:               roomID,
					allowedClients:   allowedClientIdsMap,
					connectedClients: connectedClients,
				},
				validFromUnixTime:  currentUnixTime + validFromInSecondsFromNow,
				validUntilUnixTime: currentUnixTime + validUntilInSecondsFromNow,
			}
			for _, conn := range connectedClients {
				_, userID := clientConnectionIDStringToRoomIDAndUserID(conn.ID)
				if !newRoom.IsAllowed(userID) {
					_ = conn.Close()
				}
			}
			if validFromInSecondsFromNow > 0 { // all current clients now in invalid room
				_ = newRoom.Close()
				// do not delete - room will become active later
			}

			p.rooms[roomID] = newRoom

			var response string
			activeBetweenString := " - active between " +
				time.Unix(newRoom.validFromUnixTime, 0).Format("02.01.2006-15:04:05") + " and " +
				time.Unix(newRoom.validUntilUnixTime, 0).Format("02.01.2006-15:04:05")
			if len(allowedClientIdsMap) == 0 {
				response = "added room(" + roomID + ") for all clients" + activeBetweenString
			} else {
				response = "added room(" + roomID + ") for clients" + allowedClientIdsRaw + activeBetweenString
			}
			_, _ = writer.Write([]byte(response))
			log.Println(response)
		}
		handler.HandleFunc(p.addRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.editRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.removeRoomRoute, func(writer http.ResponseWriter, request *http.Request) {
			initialParams, err := url.ParseQuery(request.URL.RawQuery)
			if err != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("could not parse url"))
				return
			}

			roomID := initialParams.Get("id")

			if len(roomID) == 0 {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("missing field in url params(string): id"))
				return
			}

			room := p.get(roomID)
			if room == nil {
				writer.WriteHeader(http.StatusNotFound)
				_, _ = writer.Write([]byte("room cannot be removed since it does not exist"))
				return
			}
			_ = room.Close()
			delete(p.rooms, roomID)

			response := "removed room(" + roomID + ")"
			_, _ = writer.Write([]byte(response))
			log.Println(response)
		})

		log.Println("Started Temporary Room Controller Server on " + p.bindAddress + ":" + strconv.Itoa(p.bindPort))
		server := http.Server{
			Addr:    p.bindAddress + ":" + strconv.Itoa(p.bindPort),
			Handler: handler,
		}
		p.raw = &server
		_ = server.ListenAndServe()
	}()

	// clean up expired rooms
	ticker := time.NewTicker(1 * time.Second) // iterate is quick, delete is rare - so 1 sec is fine
	go func() {
		for range ticker.C {
			currentUnixTime := time.Now().Unix()
			for roomID, room := range p.rooms {
				if currentUnixTime > room.validUntilUnixTime {
					_ = room.Close()
					delete(p.rooms, roomID)
					log.Println("Room " + roomID + " expired and was removed")
				}
			}
		}
	}()
}
