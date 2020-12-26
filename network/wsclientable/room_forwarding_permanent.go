package wsclientable

import (
	"gopkg.in/ini.v1"
	"log"
)

//Idea:
//     Rooms can be defined in the config file. They are immutable in their properties until the program is restarted.
//     If another controller is defined before this controller in the list given to StartRoomSignaler,
//        it takes precedence and might override the properties of rooms defined by this controller.

type PermanentRoom struct {
	Room
}

func NewPermissiblePermanentRoom(ID string) PermanentRoom {
	return NewPermanentRoom(ID, []string{})
}
func NewPermanentRoom(ID string, allowedClientIds []string) PermanentRoom {
	return PermanentRoom{Room: Room{
		ID:               ID,
		allowedClients:   createAllowedIdsMapFromSlice(allowedClientIds),
		connectedClients: make(map[string]*ClientConnection),
	}}
}

type PermanentRoomController struct {
	rooms map[string]PermanentRoom
}

func NewPermanentRoomControllerFromCFG(cfg *ini.File) *PermanentRoomController {
	var rooms []PermanentRoom
	for _, permanentRoom := range cfg.Section("permanent_rooms").ChildSections() {
		roomID := permanentRoom.Key("id").String()
		allowedClientIdsRaw := permanentRoom.Key("allowed_clients").String()
		allowedClientIdsMap := createAllowedIdsMapFromJSONArray(allowedClientIdsRaw)

		rooms = append(rooms, PermanentRoom{
			Room: Room{
				ID:               roomID,
				allowedClients:   allowedClientIdsMap,
				connectedClients: make(map[string]*ClientConnection),
			},
		})

		if len(allowedClientIdsMap) == 0 {
			log.Println("Permanent room(" + roomID + ") available for all clients")
		} else {
			log.Println("Permanent room(" + roomID + ") available for clients" + allowedClientIdsRaw)
		}
	}
	return NewPermanentRoomController(rooms...)
}
func NewPermanentRoomController(rooms ...PermanentRoom) *PermanentRoomController {
	roomsMap := make(map[string]PermanentRoom)
	for _, v := range rooms {
		roomsMap[v.ID] = v
	}
	return &PermanentRoomController{rooms: roomsMap}
}

// implement interface RoomController:

func (p *PermanentRoomController) exists(roomID string) bool {
	_, exists := p.rooms[roomID]
	return exists
}
func (p *PermanentRoomController) get(roomID string) RoomI {
	v, exists := p.rooms[roomID]
	if exists {
		return v
	}
	return nil
}
func (p *PermanentRoomController) close() error {
	for _, v := range p.rooms {
		_ = v.Close()
	}
	return nil
}
func (p *PermanentRoomController) init() {}
