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

//Idea:
//  Rooms should be temporary over a local http connection.
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

type HTTPTemporaryRoomEditor struct {
	*HTTPRoomEditor
}

func NewHTTPTemporaryRoomEditorFromCFG(cfg *ini.File) *HTTPTemporaryRoomEditor {
	bindAddress := cfg.Section("http_room_controller").Key("address").String()
	bindPort, _ := cfg.Section("http_room_controller").Key("port").Int()
	addRoomRoute := cfg.Section("http_room_controller").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_room_controller").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_room_controller").Key("remove_room_route").String()
	return NewHTTPTemporaryRoomEditorWithStorage(
		NewMutableRamRoomStorage(),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPTemporaryRoomEditorWithStorage(
	roomStorage RoomStorageI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPTemporaryRoomEditor {
	controller := NewTemporaryRoomController(roomStorage)
	return NewHTTPTemporaryRoomEditor(
		controller,
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}
func NewHTTPTemporaryRoomEditor(
	controller RoomControllerI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPTemporaryRoomEditor {
	return &HTTPTemporaryRoomEditor{
		HTTPRoomEditor: NewHTTPRoomEditor(
			controller,
			bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
		),
	}
}
func NewHTTPTemporaryRoomEditorInRam(
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPTemporaryRoomEditor {
	return NewHTTPTemporaryRoomEditorWithStorage(
		NewMutableRamRoomStorage(),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPTemporaryPersistedRoomEditorFromCfg(cfg *ini.File) *HTTPTemporaryRoomEditor {
	bindAddress := cfg.Section("http_temporary_room_controller_persisted").Key("address").String()
	bindPort, _ := cfg.Section("http_temporary_room_controller_persisted").Key("port").Int()
	addRoomRoute := cfg.Section("http_temporary_room_controller_persisted").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_temporary_room_controller_persisted").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_temporary_room_controller_persisted").Key("remove_room_route").String()
	dbPath := cfg.Section("http_temporary_room_controller_persisted").Key("db_path").String()
	return NewHTTPTemporaryRoomEditorWithStorage(
		NewTemporaryRoomBoltStorage(dbPath),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPTemporaryPersistedRoomEditor(
	bindAddress string, bindPort int,
	addRoomRoute, editRoomRoute, removeRoomRoute string,
	dbPath string,
) *HTTPTemporaryRoomEditor {
	return NewHTTPTemporaryRoomEditorWithStorage(
		NewTemporaryRoomBoltStorage(dbPath),
		bindAddress, bindPort,
		addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func (p *HTTPTemporaryRoomEditor) Init() {
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

			currentUnixTime := time.Now().Unix()

			newRoom := NewTemporaryRoom(
				roomID, UnmarshalJsonArray(allowedClientIdsRaw),
				currentUnixTime+validFromInSecondsFromNow, currentUnixTime+validUntilInSecondsFromNow,
			)
			err = p.AddRoom(roomID, newRoom, !isAddRoute)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte("error: " + err.Error()))
				return
			}

			var response string
			activeBetweenString := " - active between " +
				time.Unix(newRoom.ValidFromUnixTime, 0).Format("02.01.2006-15:04:05") + " and " +
				time.Unix(newRoom.ValidUntilUnixTime, 0).Format("02.01.2006-15:04:05")
			if len(newRoom.allowedClients) == 0 {
				response = "added temporary room(" + roomID + ") for all clients" + activeBetweenString
			} else {
				response = "added temporary room(" + roomID + ") for clients" + allowedClientIdsRaw + activeBetweenString
			}
			_, _ = writer.Write([]byte(response))
			log.Println(response)
		}
		handler.HandleFunc(p.addRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.editRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.removeRoomRoute, p.httpRemoveRoomHandleFunc)

		log.Println("Started Temporary Room Editor Server on " + p.bindAddress + ":" + strconv.Itoa(p.bindPort))
		server := http.Server{
			Addr:    p.bindAddress + ":" + strconv.Itoa(p.bindPort),
			Handler: handler,
		}
		p.raw = &server
		_ = server.ListenAndServe()
	}()
}
