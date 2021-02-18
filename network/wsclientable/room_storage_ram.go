package wsclientable

import (
	"fmt"
	"sync"
)

// RoomStorageI implementation exactly to specs.
// All data held in memory -> editable rooms lost when program exits
// No efficient implementation for ExpiredRoom cleanup - iterates and checks RoomI.IsValid

type RamRoomStorage struct {
	rooms *sync.Map // string -> RoomI
}

func NewMutableRamRoomStorage() RamRoomStorage {
	return RamRoomStorage{rooms: &sync.Map{}}
}
func NewRamRoomStorageFromSlice(rooms []PermanentRoom) RamRoomStorage {
	roomsMap := sync.Map{}
	for _, v := range rooms {
		roomsMap.Store(v.GetID(), v)
	}
	return RamRoomStorage{rooms: &roomsMap}
}
func (r RamRoomStorage) Put(room RoomI, allowOverride bool) error {
	if allowOverride {
		r.rooms.Store(room.GetID(), room)
	} else {
		_, loaded := r.rooms.LoadOrStore(room.GetID(), room)
		if loaded {
			return fmt.Errorf("room already exists")
		}
	}
	return nil
}
func (r RamRoomStorage) Remove(roomID string) (bool, error) {
	_, ok := r.rooms.Load(roomID)
	r.rooms.Delete(roomID)
	return ok, nil
}
func (r RamRoomStorage) Get(roomID string) RoomI {
	v, ok := r.rooms.Load(roomID)
	if !ok {
		return nil
	}
	return v.(RoomI)
}

func (r RamRoomStorage) Close() error {
	return nil
}

func (r RamRoomStorage) CleanExpired(removedCallback func(room *TemporaryRoom)) (*TemporaryRoom, error) {
	var nextToExpire *TemporaryRoom
	r.rooms.Range(func(key, value interface{}) bool {
		room := value.(TemporaryRoom)
		if !room.IsValid() {
			r.rooms.Delete(room.GetID())
			removedCallback(&room)
		} else if nextToExpire == nil || room.ValidUntilUnixTime < nextToExpire.ValidUntilUnixTime {
			nextToExpire = &room
		}
		return true //continue
	})
	return nextToExpire, nil
}
