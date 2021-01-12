package wsclientable

import (
	"gopkg.in/ini.v1"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

//Idea:
//  Rooms should be editable over a local http connection.
//    It is highly advisable NOT to open the port pointing to this server to the web, since there is not auth.
//  Over the port and routes specified in the config file given to the StartRoomSignaler function,
//    we can now edit otherwise permanent rooms
//    we can only remove rooms which were added by this controller
//    if we add a room that exists in another controller it seems to work - however only one room is evaluated
//    which room takes precedence depends on the order we add the controllers to the StartRoomSignaler functions
//    it is possible to edit a route, e.g. change its parameters without closing it and disconnecting all clients
//      for that either use the add edit route
//  example editing requests (python3) - note: the concrete required call depends on the config file:
//     import requests; r = requests.post("http://localhost:8087/rooms/control/add?id=test&allowed_clients=["s", "c", "parent"]"); print(r.reason, r.text)
//     import requests; r = requests.post("http://localhost:8087/rooms/control/edit?id=test&allowed_clients=["s", "c", "parent", "admin"]"); print(r.reason, r.text)
//     import requests; r = requests.post("http://localhost:8087/rooms/control/remove?id=test"); print(r.reason, r.text)
//
//NOTE: If the server is closed all created rooms will disappear and would need to be re-added...

type HTTPEditableRoomController struct {
	rooms           map[string]PermanentRoom
	raw             *http.Server
	bindAddress     string
	bindPort        int
	addRoomRoute    string
	editRoomRoute   string
	removeRoomRoute string
}

func NewHTTPEditableRoomControllerFromCFG(cfg *ini.File) *HTTPEditableRoomController {
	bindAddress := cfg.Section("http_room_controller").Key("address").String()
	bindPort, _ := cfg.Section("http_room_controller").Key("port").Int()
	addRoomRoute := cfg.Section("http_room_controller").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_room_controller").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_room_controller").Key("remove_room_route").String()
	return NewHTTPEditableRoomController(bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute)
}

func NewHTTPEditableRoomController(
	bindAddress string,
	bindPort int,
	addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPEditableRoomController {
	return &HTTPEditableRoomController{
		rooms:           make(map[string]PermanentRoom),
		raw:             nil,
		bindAddress:     bindAddress,
		bindPort:        bindPort,
		addRoomRoute:    addRoomRoute,
		editRoomRoute:   editRoomRoute,
		removeRoomRoute: removeRoomRoute,
	}
}

// implement interface RoomController:

func (p *HTTPEditableRoomController) exists(roomID string) bool {
	_, exists := p.rooms[roomID]
	return exists
}
func (p *HTTPEditableRoomController) get(roomID string) RoomI {
	v, exists := p.rooms[roomID]
	if exists {
		return v
	}
	return nil
}
func (p *HTTPEditableRoomController) close() error {
	for _, v := range p.rooms {
		_ = v.Close()
	}
	return p.raw.Close()
}

func (p *HTTPEditableRoomController) init() {
	handler := http.NewServeMux() // required for concurrent server creation used in tests...
	server := http.Server{
		Addr:    p.bindAddress + ":" + strconv.Itoa(p.bindPort),
		Handler: handler,
	}
	p.raw = &server

	go func() {
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

			allowedClientIdsMap := createAllowedIdsMapFromJSONArray(allowedClientIdsRaw)

			var connectedClients map[string]*ClientConnection
			if previousRoom, exists := p.rooms[roomID]; exists {
				if isAddRoute {
					writer.WriteHeader(http.StatusForbidden)
					_, _ = writer.Write([]byte("room exists cannot add, use edit route to edit"))
					return
				}
				connectedClients = previousRoom.connectedClients
			} else {
				connectedClients = make(map[string]*ClientConnection)
			}
			newRoom := PermanentRoom{
				Room: Room{
					ID:               roomID,
					allowedClients:   allowedClientIdsMap,
					connectedClients: connectedClients,
				},
			}
			for _, conn := range connectedClients {
				_, userID := clientConnectionIDStringToRoomIDAndUserID(conn.ID)
				if !newRoom.IsAllowed(userID) {
					_ = conn.Close()
				}
			}
			p.rooms[roomID] = newRoom

			var response string
			if len(allowedClientIdsMap) == 0 {
				response = "added room(" + roomID + ") for all clients"
			} else {
				response = "added room(" + roomID + ") for clients" + allowedClientIdsRaw
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
		log.Println("Started Http Room Controller Server on " + p.bindAddress + ":" + strconv.Itoa(p.bindPort))
		_ = server.ListenAndServe()
	}()
}
