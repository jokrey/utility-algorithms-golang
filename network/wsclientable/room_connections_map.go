package wsclientable

import (
	"sync"
)

// This class provides an in-memory, thread safe map from roomID to ConnectionMap.
// This allows to query connections in rooms (identified by roomID and userID)
// This is used, for example, to forward message between clients that only know each other by id (but within rooms)
type RoomConnectionsMap struct {
	rwMut   *sync.RWMutex
	actives map[string]*ConnectionMap
}

func NewRoomConnectionsMap() RoomConnectionsMap {
	return RoomConnectionsMap{
		rwMut:   &sync.RWMutex{},
		actives: make(map[string]*ConnectionMap),
	}
}

// Closes all connections in all rooms
func (p RoomConnectionsMap) CloseAllConnections() (int, error) {
	p.rwMut.Lock()
	defer p.rwMut.Unlock()

	counter := 0
	var err error
	for k, room := range p.actives {
		delete(p.actives, k)
		numR, e := room.CloseAll()
		counter += numR
		if e != nil {
			err = e
		}
	}
	return counter, err
}

// Closes all connections in given room
func (p RoomConnectionsMap) CloseAllInRoom(roomID string) (int, error) {
	p.rwMut.Lock()
	defer p.rwMut.Unlock()

	room, ok := p.actives[roomID]
	if ok {
		delete(p.actives, roomID)
		return room.CloseAll()
	} else {
		return 0, nil
	}
}

// Calls the given function with all connections in the given room
func (p RoomConnectionsMap) ForAllIn(roomID string, f func(connection *ClientConnection)) {
	p.rwMut.RLock()
	room, ok := p.actives[roomID]
	p.rwMut.RUnlock()

	if ok {
		room.ForAll(f)
	}
}

// calls the given function for all registered connections - keep the read lock only while iterating, delete in between is allowed
func (p *RoomConnectionsMap) ForAllRooms(f func(roomID string)) {
	p.rwMut.RLock()
	for roomID := range p.actives {
		p.rwMut.RUnlock()

		f(roomID)
		//room.ForAll(func(connection *ClientConnection) {
		//	f(roomId, connection)
		//})

		p.rwMut.RLock()
	}
	p.rwMut.RUnlock()
}

// Adds the given connection to the given room
func (p RoomConnectionsMap) AddConnectionInRoomIfNotConnected(roomID string, connection ClientConnection) bool {
	p.rwMut.RLock()
	room, exists := p.actives[roomID]
	p.rwMut.RUnlock()

	if !exists {
		p.rwMut.Lock()
		newRoom := NewConnectionMap()
		p.actives[roomID] = &newRoom
		room = &newRoom
		p.rwMut.Unlock()
	}

	return room.AddIfNotConnected(connection)
}

// Removes the given connection from the given room, removing the mapping entirely
// Return nil if no connection was removed, otherwise the removed connection is returned (IT IS NOT CLOSED YET)
func (p RoomConnectionsMap) RemoveConnectionInRoom(roomID string, userID string) *ClientConnection {
	p.rwMut.Lock()
	defer p.rwMut.Unlock()

	room, ok := p.actives[roomID]
	if ok {
		con := room.Remove(RoomIDAndUserIDToClientConnectionIDString(roomID, userID))
		if room.IsEmpty() {
			delete(p.actives, roomID)
		}
		return con
	} else {
		return nil
	}
}

// Return the connection under the given roomID, userID combination
func (p RoomConnectionsMap) GetConnectionInRoom(roomID, userID string) *ClientConnection {
	p.rwMut.RLock()
	room, ok := p.actives[roomID]
	p.rwMut.RUnlock()

	if ok {
		return room.GetByID(RoomIDAndUserIDToClientConnectionIDString(roomID, userID))
	} else {
		return nil
	}
}

// Whether the given roomID, userID combination can be queried from this map
func (p RoomConnectionsMap) IsConnected(roomID, userID string) bool {
	p.rwMut.RLock()
	room, ok := p.actives[roomID]
	p.rwMut.RUnlock()

	if ok {
		return room.IsConnected(RoomIDAndUserIDToClientConnectionIDString(roomID, userID))
	} else {
		return false
	}
}
