package wsclientable

// Room controllers are the combination of RoomStorageI and RoomConnectionsMap.
//   RoomStorage gives us the binary decision of whether a client is currently allowed in a room
//   RoomConnectionsMap gives us the addressable connection for a given id
//  Make sure to call close, always. Certain controller rely on a db and may not be usable until the process is closed

type RoomControllerI interface {
	// Initialized the controller, can be expected to have been called before any other methods
	Init()
	// Closes all underlying resources. Should be called exactly once.
	// Should make best effort to close all, even if some intermediate close operations return with an error
	Close() error

	// Returns the room under the given id
	GetRoom(roomID string) RoomI

	//Closes each connection in the room and removes the room, might allow efficient clean up of associated resources
	CloseAndRemoveRoom(roomID string) (bool, error)
	//Add or Edit the given room
	AddRoom(roomID string, newRoom RoomI, allowOverride bool) error

	// see RoomConnectionsMap.IsConnected
	IsConnected(roomID string, userID string) bool
	// see RoomConnectionsMap.AddConnectionInRoom, except the controller might acquire additional resources
	NewConnectionForRoom(roomID string, connection ClientConnection) bool
	// see RoomConnectionsMap.RemoveConnectionInRoom, except the controller might cancel additional resources
	ConnectionInRoomClosed(roomID string, userID string) *ClientConnection
	// see RoomConnectionsMap.GetConnectionInRoom
	GetConnectionInRoom(roomID, userID string) *ClientConnection
}

// Bundle of multiple controllers
// In case of duplicate roomID definitions, the order here determines which room takes precedence
type RoomControllers struct {
	controllers []RoomControllerI
}

// Bundle of multiple controllers
// In case of duplicate roomID definitions, the order here determines which room takes precedence
func BundleControllers(controllers ...RoomControllerI) RoomControllers {
	return RoomControllers{controllers}
}

func (r *RoomControllers) Init() {
	for _, v := range r.controllers {
		v.Init()
	}
}
func (r *RoomControllers) Close() error {
	var err error
	for _, v := range r.controllers {
		e := v.Close()
		if e != nil {
			err = e
		}
	}
	return err
}

func (r *RoomControllers) GetRoom(roomID string) RoomI {
	for _, v := range r.controllers {
		r := v.GetRoom(roomID)
		if r != nil {
			return r
		}
	}
	return nil
}

func (r *RoomControllers) NewConnectionForRoom(roomID string, connection ClientConnection) bool {
	for _, v := range r.controllers {
		if v.NewConnectionForRoom(roomID, connection) {
			return true
		}
	}
	return false
}
func (r *RoomControllers) ConnectionInRoomClosed(roomID string, userID string) *ClientConnection {
	for _, v := range r.controllers {
		con := v.ConnectionInRoomClosed(roomID, userID)
		if con != nil {
			return con
		}
	}
	return nil
}
func (r *RoomControllers) GetConnectionInRoom(roomID, userID string) *ClientConnection {
	for _, v := range r.controllers {
		c := v.GetConnectionInRoom(roomID, userID)
		if c != nil {
			return c
		}
	}
	return nil
}
func (r *RoomControllers) IsConnected(roomID string, userID string) bool {
	for _, v := range r.controllers {
		if v.IsConnected(roomID, userID) {
			return true
		}
	}
	return false
}

func (r *RoomControllers) CloseAndRemoveRoom(roomID string) (bool, error) {
	for _, v := range r.controllers {
		existed, e := v.CloseAndRemoveRoom(roomID)
		if existed || e != nil {
			return existed, e
		}
	}
	return false, nil
}

func (r *RoomControllers) AddRoom(_ string, _ RoomI, _ bool) error {
	panic("not implemented for RoomControllers, cannot decide which room controller to use - call specific directly")
}
