package wsclientable

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
	"time"
)

//IDEA:
//  RepeatingRooms can be held in a database.
//  Upon query the appropriate room will be decoded from the database and checked for validity
//
// Only works for RepeatingRoom. When adding anything else, this code will panic.

type BoltRepeatingRoomStorage struct {
	db *bolt.DB
}

func NewRepeatingRoomBoltStorage(dbPath string) BoltRepeatingRoomStorage {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		panic(err)
	}

	return BoltRepeatingRoomStorage{db: db}
}

func (b BoltRepeatingRoomStorage) Close() error {
	return b.db.Close()
}
func (b BoltRepeatingRoomStorage) Put(roomI RoomI, allowOverride bool) error {
	room := roomI.(RepeatingRoom)
	return b.db.Update(func(tx *bolt.Tx) error {
		roomIDBytes := []byte(room.GetID())

		var err error
		var roomB *bolt.Bucket
		if allowOverride {
			roomB, err = tx.CreateBucketIfNotExists(roomIDBytes)
			if err != nil {
				return err
			} //auto rollback
		} else {
			roomB, err = tx.CreateBucket(roomIDBytes)
			if err != nil {
				if err == bolt.ErrBucketExists {
					return fmt.Errorf("room already exists")
				}
				return err
			} //auto rollback
		}
		err = encodeRepeatingRoomIntoBucket(room, roomB)
		if err != nil {
			return err
		} //auto rollback

		return nil
	})
}

func (b BoltRepeatingRoomStorage) Remove(roomID string) (bool, error) {
	previouslyExisted := true
	err := b.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(roomID))
		if err == bolt.ErrBucketNotFound {
			previouslyExisted = false
			return nil
		}
		return err
	})
	return previouslyExisted, err
}

func (b BoltRepeatingRoomStorage) Get(roomID string) RoomI {
	var decodedRoom RoomI
	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(roomID))
		if b == nil {
			decodedRoom = nil
			return nil
		}

		decodedRoom = decodeRepeatingRoomFromBucket(roomID, b)

		return nil
	})
	if err != nil {
		log.Printf("database failed Get, returning nil - err: %v", err)
		return nil
	}
	return decodedRoom
}

//bucket must be in a Update context
func encodeRepeatingRoomIntoBucket(room RepeatingRoom, roomB *bolt.Bucket) error {
	allowedClientsB, err := roomB.CreateBucketIfNotExists([]byte("allowedClientIDs"))
	if err != nil {
		return err
	} //auto rollback
	for allowedClientName := range room.allowedClients {
		err = allowedClientsB.Put([]byte(allowedClientName), []byte{})
		if err != nil {
			return err
		} //auto rollback
	}

	err = roomB.Put([]byte("FirstTimeUnixTimestamp"), int64ToBytes(room.FirstTimeUnixTimestamp))
	if err != nil {
		return err
	} //auto rollback
	err = roomB.Put([]byte("RepeatEverySeconds"), int64ToBytes(room.RepeatEverySeconds))
	if err != nil {
		return err
	} //auto rollback
	err = roomB.Put([]byte("DurationInSeconds"), int64ToBytes(room.DurationInSeconds))
	if err != nil {
		return err
	} //auto rollback

	return nil //success, no error
}

//bucket must be in at least a View context
func decodeRepeatingRoomFromBucket(roomID string, roomB *bolt.Bucket) RepeatingRoom {
	var allowedClientIds []string
	allowedClientsB := roomB.Bucket([]byte("allowedClientIDs"))
	c := allowedClientsB.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		allowedClientId := string(k)
		allowedClientIds = append(allowedClientIds, allowedClientId)
	}

	firstTimeUnixTimestamp := int64FromBytes(roomB.Get([]byte("FirstTimeUnixTimestamp")))
	repeatEverySeconds := int64FromBytes(roomB.Get([]byte("RepeatEverySeconds")))
	durationInSeconds := int64FromBytes(roomB.Get([]byte("DurationInSeconds")))

	return RepeatingRoom{
		Room: Room{
			ID:             roomID,
			allowedClients: CreateAllowedIdsMapFromSlice(allowedClientIds),
		},
		FirstTimeUnixTimestamp: firstTimeUnixTimestamp,
		RepeatEverySeconds:     repeatEverySeconds,
		DurationInSeconds:      durationInSeconds,
	}
}
