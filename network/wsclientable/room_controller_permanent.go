package wsclientable

import (
	"gopkg.in/ini.v1"
	"log"
	"strings"
)

//Idea:
//     Rooms can be defined in the config file. They are immutable in their properties until the program is restarted.
//     If another controller is defined before this controller in the list given to StartRoomSignaler,
//        it takes precedence and might override the properties of rooms defined by this controller.

func NewPermanentRoomControllerFromCFG(cfg *ini.File) *EditableRoomController {
	var rooms []PermanentRoom
	for _, permanentRoom := range cfg.Section("permanent_rooms").ChildSections() {
		roomID := permanentRoom.Key("id").String()
		allowedClientIds := UnmarshalJsonArray(permanentRoom.Key("allowed_clients").String())

		rooms = append(rooms, NewPermanentRoom(roomID, allowedClientIds))

		if len(allowedClientIds) == 0 {
			log.Println("Permanent room(" + roomID + ") available for all clients")
		} else {
			log.Println("Permanent room(" + roomID + ") available for clients" + strings.Join(allowedClientIds, ", "))
		}
	}
	return NewPermanentRoomController(rooms...)
}
func NewPermanentRoomController(rooms ...PermanentRoom) *EditableRoomController {
	return &EditableRoomController{
		RoomConnectionsMap: NewRoomConnectionsMap(),
		store:              NewRamRoomStorageFromSlice(rooms),
	}
}
