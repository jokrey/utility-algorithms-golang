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
//    we can only Remove rooms which were added by this controller
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

type HTTPRoomEditor struct {
	RoomControllerI
	raw             *http.Server
	bindAddress     string
	bindPort        int
	addRoomRoute    string
	editRoomRoute   string
	removeRoomRoute string
}

func NewHTTPRoomEditor(
	controller RoomControllerI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPRoomEditor {
	return &HTTPRoomEditor{
		RoomControllerI: controller,
		raw:             nil,
		bindAddress:     bindAddress,
		bindPort:        bindPort,
		addRoomRoute:    addRoomRoute,
		editRoomRoute:   editRoomRoute,
		removeRoomRoute: removeRoomRoute,
	}
}

func NewHTTPRoomEditorFromCFG(cfg *ini.File) *HTTPRoomEditor {
	bindAddress := cfg.Section("http_room_controller").Key("address").String()
	bindPort, _ := cfg.Section("http_room_controller").Key("port").Int()
	addRoomRoute := cfg.Section("http_room_controller").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_room_controller").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_room_controller").Key("remove_room_route").String()
	return NewHTTPRoomEditorWithStorage(
		NewMutableRamRoomStorage(),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPRoomEditorWithStorage(
	roomStorage RoomStorageI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPRoomEditor {
	controller := NewEditableRoomController(roomStorage)
	return NewHTTPRoomEditor(
		&controller,
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}
func NewHTTPRoomEditorInRam(
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPRoomEditor {
	return NewHTTPRoomEditorWithStorage(
		NewMutableRamRoomStorage(),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

// implement interface RoomControllerI:
func (p *HTTPRoomEditor) Close() error {
	e1 := p.RoomControllerI.Close()
	e2 := p.raw.Close()
	if e1 != nil {
		return e1
	}
	return e2
}

func (p *HTTPRoomEditor) Init() {
	go func() {
		handler := http.NewServeMux() // required for concurrent server creation used in tests...
		server := http.Server{
			Addr:    p.bindAddress + ":" + strconv.Itoa(p.bindPort),
			Handler: handler,
		}
		p.raw = &server

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

			newRoom := NewPermanentRoom(roomID, UnmarshalJsonArray(allowedClientIdsRaw))
			err = p.AddRoom(roomID, newRoom, !isAddRoute)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte("error: " + err.Error()))
				return
			}

			var response string
			if len(newRoom.allowedClients) == 0 {
				response = "added permanent room(" + roomID + ") for all clients"
			} else {
				response = "added permanent room(" + roomID + ") for clients" + allowedClientIdsRaw
			}
			_, _ = writer.Write([]byte(response))
			log.Println(response)
		}
		handler.HandleFunc(p.addRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.editRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.removeRoomRoute, p.httpRemoveRoomHandleFunc)
		log.Println("Started Http Room Editor Server on " + p.bindAddress + ":" + strconv.Itoa(p.bindPort))
		_ = server.ListenAndServe()
	}()
}

func (p *HTTPRoomEditor) httpRemoveRoomHandleFunc(writer http.ResponseWriter, request *http.Request) {
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

	existed, err := p.CloseAndRemoveRoom(roomID)
	if !existed {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte("cannot remove room, not found"))
		return
	}
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("error closing room: " + err.Error()))
		return
	}

	response := "removed room(" + roomID + ")"
	_, _ = writer.Write([]byte(response))
	log.Println(response)
}
