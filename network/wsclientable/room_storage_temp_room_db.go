package wsclientable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"log"
	"time"
)

//IDEA:
//  Temporary Rooms can be held in a database.
//  Upon query the appropriate room will be decoded from the database and checked for validity
//
//  Database Model ('-' denotes bolt buckets, '->' denotes key-value pair):
//     - "rooms": (MANY sub buckets)
//        - <roomID>
//        	- "allowedClientIDs" (SET, likely small)
//            <ClientID-1> -> ""
//            <ClientID-2> -> ""
//			  etc...
//          "ValidFromUnixTime" -> <ValidFromUnixTime>
//          "ValidUntilUnixTime" -> <ValidUntilUnixTime>
//     - "expirations" (MANY sub buckets (SORTED, i.e. earliest first))
//		  - <ValidUntilUnixTime> (SET, likely small)
//          <roomID-1> -> ""
//          <roomID-2> -> ""
//          etc...
//    This model allows:
//    	efficient access to room data via roomID
//      efficient access to earliest expiration, early stopping when exceeded now (by bytes level comparison)
//
// Only works for TemporaryRooms. When adding anything else, this code will panic.

type TemporaryRoomStorageI interface {
	RoomStorageI

	// Cleans all expired rooms
	// (based on RoomI.IsValid OR some specific logic in case the room storage is specific to a room-type)
	// Returns the next room that will expire or nil and an error
	CleanExpired(removedCallback func(*TemporaryRoom)) (*TemporaryRoom, error)
}

type BoltTemporaryRoomStorage struct {
	db *bolt.DB
}

func NewTemporaryRoomBoltStorage(dbPath string) BoltTemporaryRoomStorage {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("rooms"))       //stores the raw room data (map roomID -> roomData)
		_, err = tx.CreateBucketIfNotExists([]byte("expirations")) //stores the expirationDate mapped to roomID
		if err != nil {
			return err
		} //auto rollback
		return nil
	})
	if err != nil {
		panic(err)
	}

	return BoltTemporaryRoomStorage{db: db}
}

func (b BoltTemporaryRoomStorage) Close() error {
	return b.db.Close()
}
func (b BoltTemporaryRoomStorage) Put(roomI RoomI, allowOverride bool) error {
	room := roomI.(TemporaryRoom)
	return b.db.Update(func(tx *bolt.Tx) error {
		roomIDBytes := []byte(room.GetID())
		roomsB := tx.Bucket([]byte("rooms"))
		expirationsB := tx.Bucket([]byte("expirations"))

		roomB := roomsB.Bucket(roomIDBytes)
		if roomB != nil {
			if allowOverride {
				_, err := removeTemporaryRoomWithExpiration(roomIDBytes, roomsB, expirationsB)
				if err != nil {
					return err
				} //auto rollback
			} else {
				return fmt.Errorf("room already exists")
			}
		}

		roomB, err := roomsB.CreateBucket(roomIDBytes)
		err = encodeTemporaryRoomIntoBucket(room, roomB)
		if err != nil {
			return err
		} //auto rollback

		expirationB, err := expirationsB.CreateBucketIfNotExists(int64ToBytes(room.ValidUntilUnixTime))
		if err != nil {
			return err
		} //auto rollback
		err = encodeIntoExpiration(room, expirationB)
		if err != nil {
			return err
		} //auto rollback

		return nil
	})
}

func (b BoltTemporaryRoomStorage) Remove(roomID string) (bool, error) {
	previouslyExisted := true
	err := b.db.Update(func(tx *bolt.Tx) error {
		existed, err := removeTemporaryRoomWithExpiration([]byte(roomID), tx.Bucket([]byte("rooms")), tx.Bucket([]byte("expirations")))
		previouslyExisted = existed
		return err
	})
	return previouslyExisted, err
}

//must be in UPDATE context
func removeTemporaryRoomWithExpiration(roomIDBytes []byte, roomsB, expirationsB *bolt.Bucket) (bool, error) {
	roomB := roomsB.Bucket(roomIDBytes)
	if roomB == nil { //does not exist
		return false, nil
	}
	decodedRoom := decodeTemporaryRoomFromBucket("", roomB) //roomID does not matter, never looked at
	err := roomsB.DeleteBucket(roomIDBytes)
	if err != nil && err != bolt.ErrBucketNotFound {
		return true, err
	} //auto rollback

	expirationB := expirationsB.Bucket(int64ToBytes(decodedRoom.ValidUntilUnixTime))
	err = expirationB.Delete(roomIDBytes)
	if err != nil {
		return true, err
	} //auto rollback

	return true, nil
}

func (b BoltTemporaryRoomStorage) Get(roomID string) RoomI {
	var decodedRoom RoomI
	err := b.db.View(func(tx *bolt.Tx) error {
		roomsB := tx.Bucket([]byte("rooms"))
		b := roomsB.Bucket([]byte(roomID))
		if b == nil {
			decodedRoom = nil
			return nil
		}

		decodedRoom = decodeTemporaryRoomFromBucket(roomID, b)

		return nil
	})
	if err != nil {
		log.Printf("database failed Get, returning nil - err: %v", err)
		return nil
	}
	return decodedRoom
}

//func (b BoltTemporaryRoomStorage) GetNextExpiration() *TemporaryRoom {
//	var decodedRoom *TemporaryRoom
//	err := b.db.Update(func(tx *bolt.Tx) error {
//		roomsB := tx.Bucket([]byte("rooms"))
//
//		expirationsB := tx.Bucket([]byte("expirations"))
//		expirationsC := expirationsB.Cursor()
//
//		k, _ := expirationsC.First()
//		expirationB := expirationsB.Bucket(k)
//		if expirationB != nil {
//			nextExpiringRoomIDBytes := decodeFirstExpiredRoomIDBytes(expirationB)
//			roomB := roomsB.Bucket(nextExpiringRoomIDBytes)
//			decodedRoom = decodeTemporaryRoomFromBucket(string(nextExpiringRoomIDBytes), roomB)
//		}
//
//		return nil
//	})
//	if err != nil {
//		log.Printf("database failed Get, returning nil - err: %v", err)
//		return nil
//	}
//	return decodedRoom
//}

func (b BoltTemporaryRoomStorage) CleanExpired(removedCallback func(*TemporaryRoom)) (*TemporaryRoom, error) {
	var nextToExpire TemporaryRoom
	err := b.db.Update(func(tx *bolt.Tx) error {
		roomsB := tx.Bucket([]byte("rooms"))

		nowBytes := int64ToBytes(time.Now().Unix())
		expirationsB := tx.Bucket([]byte("expirations"))
		expirationsC := expirationsB.Cursor()

		// Iterate from first until now - expect fast iterate and rare delete
		var k, v []byte
		for k, v = expirationsC.First(); k != nil && bytes.Compare(k, nowBytes) <= 0; k, v = expirationsC.Next() {
			if v != nil {
				log.Fatal("if v == nil, value at k is a bucket... This is what we expect here")
			}

			expirationTime := int64FromBytes(k)
			expirationB := expirationsB.Bucket(k)
			expiredRoomIDsBytes := decodeExpiredRoomIDsBytes(expirationB)
			for _, roomIDBytes := range expiredRoomIDsBytes {
				roomB := roomsB.Bucket(roomIDBytes)
				decodedRoom := decodeTemporaryRoomFromBucket(string(roomIDBytes), roomB)
				if decodedRoom.ValidUntilUnixTime == expirationTime { //only delete if the decoded bucket is still the same
					err := roomsB.DeleteBucket(roomIDBytes)
					if err != nil {
						return err
					} //auto rollback
					removedCallback(&decodedRoom)
				}
			}
			err := expirationsB.DeleteBucket(k)
			if err != nil {
				return err
			} //auto rollback
		}

		// k is here the very last decoded, that failed the loop check - can also be nil
		if k != nil {
			expirationB := expirationsB.Bucket(k)
			if expirationB != nil {
				nextExpiringRoomIDBytes := decodeFirstExpiredRoomIDBytes(expirationB)
				roomB := roomsB.Bucket(nextExpiringRoomIDBytes)
				if roomB != nil {
					nextToExpire = decodeTemporaryRoomFromBucket(string(nextExpiringRoomIDBytes), roomB)
				}
			}
		}

		return nil
	})
	return &nextToExpire, err
}

func (b BoltTemporaryRoomStorage) logContents() error {
	return b.db.View(func(tx *bolt.Tx) error {
		prettyPrint(tx)

		return nil
	})
}

func prettyPrint(tx *bolt.Tx) {
	c := tx.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if v == nil { //is bucket
			log.Println("- ", string(k))
			prettyPrintInner(1, tx.Bucket(k))
		} else {
			log.Println(string(k), " -> ", string(v))
		}
	}
}
func prettyPrintInner(indent int, b *bolt.Bucket) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += " "
	}
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if v == nil { //is bucket
			log.Println(indentStr, "- ", string(k))
			prettyPrintInner(indent+1, b.Bucket(k))
		} else {
			log.Println(indentStr, string(k), " -> ", string(v))
		}
	}
}

//bucket must be in at least a View context
func decodeExpiredRoomIDsBytes(expirationB *bolt.Bucket) [][]byte {
	var expiredRoomIDsBytes [][]byte
	c := expirationB.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		roomIDBytes := k
		expiredRoomIDsBytes = append(expiredRoomIDsBytes, roomIDBytes)
	}
	return expiredRoomIDsBytes
}

//bucket must be in at least a View context
func decodeFirstExpiredRoomIDBytes(expirationB *bolt.Bucket) []byte {
	k, _ := expirationB.Cursor().First()
	return k
}

//bucket must be in a Update context
func encodeIntoExpiration(room TemporaryRoom, expirationB *bolt.Bucket) error {
	return expirationB.Put([]byte(room.GetID()), []byte{})
}

//bucket must be in a Update context
func encodeTemporaryRoomIntoBucket(room TemporaryRoom, roomB *bolt.Bucket) error {
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

	err = roomB.Put([]byte("ValidFromUnixTime"), int64ToBytes(room.ValidFromUnixTime))
	if err != nil {
		return err
	} //auto rollback
	err = roomB.Put([]byte("ValidUntilUnixTime"), int64ToBytes(room.ValidUntilUnixTime))
	if err != nil {
		return err
	} //auto rollback

	return nil //success, no error
}

//bucket must be in at least a View context
func decodeTemporaryRoomFromBucket(roomID string, roomB *bolt.Bucket) TemporaryRoom {
	var allowedClientIds []string
	allowedClientsB := roomB.Bucket([]byte("allowedClientIDs"))
	c := allowedClientsB.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		allowedClientId := string(k)
		allowedClientIds = append(allowedClientIds, allowedClientId)
	}

	validFromUnixTime := int64FromBytes(roomB.Get([]byte("ValidFromUnixTime")))
	validUntilUnixTime := int64FromBytes(roomB.Get([]byte("ValidUntilUnixTime")))

	return TemporaryRoom{
		Room: Room{
			ID:             roomID,
			allowedClients: CreateAllowedIdsMapFromSlice(allowedClientIds),
		},
		ValidFromUnixTime:  validFromUnixTime,
		ValidUntilUnixTime: validUntilUnixTime,
	}
}

func int64ToBytes(i int64) []byte {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(i))
	return bs
}
func int64FromBytes(bytes []byte) int64 {
	return int64(binary.BigEndian.Uint64(bytes))
}
