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
//  Rooms should be repeating over a local http connection.
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

type HTTPRepeatingRoomEditor struct {
	*HTTPRoomEditor
}

func NewHTTPRepeatingRoomEditor(
	controller RoomControllerI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPRepeatingRoomEditor {
	return &HTTPRepeatingRoomEditor{
		HTTPRoomEditor: NewHTTPRoomEditor(
			controller,
			bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
		),
	}
}

func NewHTTPRepeatingRoomEditorFromCFG(cfg *ini.File) *HTTPRepeatingRoomEditor {
	bindAddress := cfg.Section("http_repeating_room_controller").Key("address").String()
	bindPort, _ := cfg.Section("http_repeating_room_controller").Key("port").Int()
	addRoomRoute := cfg.Section("http_repeating_room_controller").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_repeating_room_controller").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_repeating_room_controller").Key("remove_room_route").String()
	return NewHTTPRepeatingRoomEditorWithStorage(
		NewMutableRamRoomStorage(),
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPRepeatingRoomEditorWithStorage(
	roomStorage RoomStorageI,
	bindAddress string, bindPort int, addRoomRoute, editRoomRoute, removeRoomRoute string,
) *HTTPRepeatingRoomEditor {
	controller := NewRepeatingRoomController(roomStorage)
	return NewHTTPRepeatingRoomEditor(
		controller,
		bindAddress, bindPort, addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPRepeatingRoomPersistingEditorFromCFG(cfg *ini.File) *HTTPRepeatingRoomEditor {
	bindAddress := cfg.Section("http_repeating_room_controller_persisted").Key("address").String()
	bindPort, _ := cfg.Section("http_repeating_room_controller_persisted").Key("port").Int()
	addRoomRoute := cfg.Section("http_repeating_room_controller_persisted").Key("add_room_route").String()
	editRoomRoute := cfg.Section("http_repeating_room_controller_persisted").Key("edit_room_route").String()
	removeRoomRoute := cfg.Section("http_repeating_room_controller_persisted").Key("remove_room_route").String()
	dbPath := cfg.Section("http_repeating_room_controller_persisted").Key("db_path").String()
	return NewHTTPRepeatingRoomEditorWithStorage(
		NewRepeatingRoomBoltStorage(dbPath),
		bindAddress, bindPort,
		addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func NewHTTPRepeatingPersistedRoomEditor(
	bindAddress string, bindPort int,
	addRoomRoute, editRoomRoute, removeRoomRoute string,
	dbPath string,
) *HTTPRepeatingRoomEditor {
	return NewHTTPRepeatingRoomEditorWithStorage(
		NewRepeatingRoomBoltStorage(dbPath),
		bindAddress, bindPort,
		addRoomRoute, editRoomRoute, removeRoomRoute,
	)
}

func (p HTTPRepeatingRoomEditor) Init() {
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

			firstTimeUnixInSecondsFromNow, errF := strconv.ParseInt(initialParams.Get("first_time_unix_in_seconds_from_now"), 10, 64)
			repeatEverySeconds, errR := strconv.ParseInt(initialParams.Get("repeat_every_seconds"), 10, 64)
			durationInSeconds, errU := strconv.ParseInt(initialParams.Get("duration_in_seconds"), 10, 64)

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
				_, _ = writer.Write([]byte("field in url params(missing or not a number): first_time_unix_in_seconds_from_now"))
				return
			}
			if errR != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("field in url params(missing or not a number): repeat_every_seconds"))
				return
			}
			if errU != nil {
				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write([]byte("field in url params(missing or not a number): duration_in_seconds"))
				return
			}

			currentUnixTime := time.Now().Unix()

			newRoom := NewRepeatingRoom(
				roomID, UnmarshalJsonArray(allowedClientIdsRaw),
				currentUnixTime+firstTimeUnixInSecondsFromNow, repeatEverySeconds, durationInSeconds,
			)
			err = p.AddRoom(roomID, newRoom, !isAddRoute)
			if err != nil {
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte("error: " + err.Error()))
				return
			}

			var response string
			if len(newRoom.allowedClients) == 0 {
				response = "added repeating room(" + roomID + ") for all clients"
			} else {
				response = "added repeating room(" + roomID + ") for clients " + allowedClientIdsRaw +
					" at first-at=" + time.Unix(newRoom.FirstTimeUnixTimestamp, 0).String() +
					", duration=" + (time.Duration(newRoom.DurationInSeconds) * time.Second).String() +
					", repeating=" + (time.Duration(newRoom.RepeatEverySeconds) * time.Second).String()
			}
			_, _ = writer.Write([]byte(response))
			log.Println(response)
		}
		handler.HandleFunc(p.addRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.editRoomRoute, addOrEditRouteFunc)
		handler.HandleFunc(p.removeRoomRoute, p.httpRemoveRoomHandleFunc)

		log.Println("Started Repeating Room Editor Server on " + p.bindAddress + ":" + strconv.Itoa(p.bindPort))
		server := http.Server{
			Addr:    p.bindAddress + ":" + strconv.Itoa(p.bindPort),
			Handler: handler,
		}
		p.raw = &server
		_ = server.ListenAndServe()
	}()
}
