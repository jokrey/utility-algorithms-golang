package wsclientable

// Minimal RoomControllerI implementation
type EditableRoomController struct {
	RoomConnectionsMap
	store RoomStorageI
}

func NewEditableRoomControllerInRam() EditableRoomController {
	return NewEditableRoomController(NewMutableRamRoomStorage())
}
func NewEditableRoomController(roomStorage RoomStorageI) EditableRoomController {
	return EditableRoomController{
		RoomConnectionsMap: NewRoomConnectionsMap(),
		store:              roomStorage,
	}
}

// implement interface RoomControllerI:
func (p *EditableRoomController) GetRoom(roomID string) RoomI {
	return p.store.Get(roomID)
}
func (p *EditableRoomController) Close() error {
	_, e1 := p.CloseAllConnections()
	e2 := p.store.Close()
	if e1 != nil {
		return e1
	}
	return e2
}

func (p *EditableRoomController) NewConnectionForRoom(roomID string, connection ClientConnection) bool {
	room := p.GetRoom(roomID)
	_, userID, e := ConnectionIDStringToRoomIDAndUserID(connection.ID)
	if e == nil && room != nil && room.IsAllowed(userID) {
		return p.AddConnectionInRoomIfNotConnected(roomID, connection)
	}
	return false
}
func (p *EditableRoomController) ConnectionInRoomClosed(roomID string, userID string) *ClientConnection {
	return p.RemoveConnectionInRoom(roomID, userID)
}

func (p *EditableRoomController) CloseAndRemoveRoom(roomID string) (bool, error) {
	_, e1 := p.CloseAllInRoom(roomID)
	existed, e2 := p.store.Remove(roomID)
	if e1 != nil {
		return existed, e1
	}
	return existed, e2
}

func (p *EditableRoomController) AddRoom(roomID string, newRoom RoomI, allowOverride bool) error {
	err := p.store.Put(newRoom, allowOverride)
	if err != nil {
		return err
	}
	p.ForAllIn(roomID, func(connection *ClientConnection) {
		_, userID, e := ConnectionIDStringToRoomIDAndUserID(connection.ID)
		if e != nil || !newRoom.IsAllowed(userID) {
			_ = connection.Close() //automatically removes connection in room also
		}
	})
	return nil
}

func (p *EditableRoomController) Init() {}
