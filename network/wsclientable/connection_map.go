package wsclientable

import (
	"sync"
)

// This class provides an in-memory, thread safe map from connectionID to ClientConnection.
// This allows to query a connection by id and send data to it.
// This is used, for example, to forward message between clients that only know each other by id
type ConnectionMap struct {
	rwMut *sync.RWMutex
	rMap  map[string]*ClientConnection
}

func NewConnectionMap() ConnectionMap {
	return ConnectionMap{
		rwMut: &sync.RWMutex{},
		rMap:  make(map[string]*ClientConnection),
	}
}

// adds and returns true when the connection was newly added
// if this function returns false, the given connection was NOT added to the map and should be closed (id collision)
func (m ConnectionMap) AddIfNotConnected(connection ClientConnection) bool {
	m.rwMut.Lock()
	defer m.rwMut.Unlock()

	_, exists := m.rMap[connection.ID]
	if exists {
		return false
	}
	m.rMap[connection.ID] = &connection
	return true
}

// Removes the connection with the given id from this map
// Return nil if no connection was removed, otherwise the removed connection is returned (IT IS NOT CLOSED YET)
func (m ConnectionMap) Remove(connectionID string) *ClientConnection {
	m.rwMut.Lock()
	defer m.rwMut.Unlock()

	con := m.rMap[connectionID]
	delete(m.rMap, connectionID)
	return con
}

// Returns the connection with the id or nil if it does not exist in the map
func (m ConnectionMap) GetByID(connectionID string) *ClientConnection {
	m.rwMut.RLock()
	defer m.rwMut.RUnlock()

	v, exists := m.rMap[connectionID]
	if !exists {
		return nil
	}
	return v
}

// Calls the given function for all connections in the map
func (m ConnectionMap) ForAll(f func(connection *ClientConnection)) {
	m.rwMut.RLock()
	for _, v := range m.rMap {
		m.rwMut.RUnlock()
		f(v) //can acquire x lock, now that r is unlocked
		m.rwMut.RLock()
	}
	m.rwMut.RUnlock()
}

// Iterates the map and closes all connection, returns the latest error
// (i.e. if there are multiple errors, the method will continue to iterate and return only the latest error)
func (m ConnectionMap) CloseAll() (int, error) {
	num := len(m.rMap)
	var err error
	m.ForAll(func(connection *ClientConnection) {
		e := connection.Close()
		if e != nil {
			err = e
		}
	})
	return num, err
}

// Whether the map is empty
func (m ConnectionMap) IsEmpty() bool {
	m.rwMut.RLock()
	defer m.rwMut.RUnlock()

	return len(m.rMap) == 0
}

// Whether the map contains a connection with the given id
func (m ConnectionMap) IsConnected(connectionID string) bool {
	m.rwMut.RLock()
	defer m.rwMut.RUnlock()

	_, exists := m.rMap[connectionID]
	return exists
}
