package wsclientable

import (
	"log"
	"time"
)

// Base Interface for all rooms
// Rooms have an ID
// Rooms can allow or reject clients (which may depend on the connectionID itself or context factors such as time)
// Rooms can become Invalid, in which case the would and should be removed by RoomStorageI.CleanExpired
type RoomI interface {
	// The ID of this room
	GetID() string
	// Whether the given connectionID is currently allowed in this room. If not, the connection to the client should be closed
	IsAllowed(userID string) bool
	// Whether the room is still valid. If this returns false, IsAllowed must return false also
	IsValid() bool
}

// Room base struct.
// Stores its own roomID(ID)
// Stores all allowedClients, if the list is empty ALL connectionIDs are allowed
type Room struct {
	ID             string
	allowedClients map[string]bool // always true, just used because somehow go does not support search in slice - IMMUTABLE
}

// A PermanentRoom is a Room, that will always return true for RoomI.IsValid
type PermanentRoom struct {
	Room
}

// Create PermanentRoom that allows clients with any ID
func NewPermissiblePermanentRoom(ID string) PermanentRoom {
	return NewPermanentRoom(ID, []string{})
}

// Create PermanentRoom that allows only clients with one of the given IDs
func NewPermanentRoom(ID string, allowedClientIds []string) PermanentRoom {
	return PermanentRoom{Room: Room{
		ID:             ID,
		allowedClients: CreateAllowedIdsMapFromSlice(allowedClientIds),
	}}
}

func (r PermanentRoom) GetID() string {
	return r.ID
}
func (r PermanentRoom) IsAllowed(userID string) bool {
	if len(r.allowedClients) == 0 {
		return true
	}

	_, ok := r.allowedClients[userID]
	return ok
}
func (r PermanentRoom) IsValid() bool {
	return true
}

// A temporary room is a room, but only allows clients in the specified time frame
//   There is NO functionality to close a room when the time is up
//   For that the RoomStorageI.CleanExpired method has to be called - and connections closed via the callback
//   That is left to the controller, because naturally that is the only class with enough information to do so
//!  However it is very desirable that the RoomStorageI.CleanExpired method has an efficient implementation that scales
type TemporaryRoom struct {
	Room
	ValidFromUnixTime  int64
	ValidUntilUnixTime int64
}

func (r TemporaryRoom) GetID() string {
	return r.ID
}
func (r TemporaryRoom) IsAllowed(userID string) bool {
	currentUnixTime := time.Now().Unix()
	timeSlotOk := currentUnixTime >= r.ValidFromUnixTime && currentUnixTime <= r.ValidUntilUnixTime
	if len(r.allowedClients) == 0 {
		return timeSlotOk
	}

	_, ok := r.allowedClients[userID]
	return ok && timeSlotOk
}
func (r TemporaryRoom) IsValid() bool {
	currentUnixTime := time.Now().Unix()
	return currentUnixTime <= r.ValidUntilUnixTime
}
func NewTemporaryRoom(ID string, allowedClientIds []string, validFromUnixTime, validUntilUnixTime int64) TemporaryRoom {
	return TemporaryRoom{
		Room{ID, CreateAllowedIdsMapFromSlice(allowedClientIds)},
		validFromUnixTime,
		validUntilUnixTime,
	}
}

// A repeating room is a room, but only allows clients in the specified time frame
//   There is NO functionality to close a room when the time is up
//   For that the RoomStorageI.CleanExpired method has to be called - and connections closed via the callback
//   That is left to the controller, because naturally that is the only class with enough information to do so
//!  However it is very desirable that the RoomStorageI.CleanExpired method has an efficient implementation that scales
type RepeatingRoom struct {
	Room
	FirstTimeUnixTimestamp int64
	RepeatEverySeconds     int64
	DurationInSeconds      int64
}

func (r RepeatingRoom) GetID() string {
	return r.ID
}
func (r RepeatingRoom) IsAllowed(userID string) bool {
	now := time.Now().Unix()
	if now < r.FirstTimeUnixTimestamp {
		return false
	}

	timeSlotOk := Pmod(now-r.FirstTimeUnixTimestamp, r.RepeatEverySeconds) <= r.DurationInSeconds //meth
	if len(r.allowedClients) == 0 {
		return timeSlotOk
	}

	log.Printf("Repeating Room Is Allowed - timeslot ok: %v", timeSlotOk)

	_, ok := r.allowedClients[userID]

	log.Printf("Repeating Room in allowed rooms: %v, userID=%v, r.allowedClients=%v", ok, userID, r.allowedClients)
	return ok && timeSlotOk
}
func Pmod(a, b int64) int64 {
	return (a%b + b) % b
}
func (r RepeatingRoom) IsValid() bool {
	return true
}

// Create PermanentRoom that allows only clients with one of the given IDs
func NewRepeatingRoom(ID string, allowedClientIds []string, firstTimeUnixTimestamp, repeatEverySeconds, durationInSeconds int64) RepeatingRoom {
	return RepeatingRoom{
		Room: Room{
			ID:             ID,
			allowedClients: CreateAllowedIdsMapFromSlice(allowedClientIds),
		},
		FirstTimeUnixTimestamp: firstTimeUnixTimestamp,
		RepeatEverySeconds:     repeatEverySeconds,
		DurationInSeconds:      durationInSeconds,
	}
}

// Room Storage is the transparent interface to access rooms based on their roomID.
// Mutation has to be supported by any implementation
// Used to switch out the underlying storage location in RoomControllers

type RoomStorageI interface {
	// Puts the given room into storage - if already exists: override
	Put(room RoomI, allowOverride bool) error
	// Remove room if it exists(return (true, nil), otherwise return (false, nil)
	// If the underlying storage implementation fails, it may return an error
	Remove(roomID string) (bool, error)
	// Returns nil if room does not exist
	Get(roomID string) RoomI

	// Closes underlying resources, should only be called once. Has to be called (can be deferred).
	Close() error
}
