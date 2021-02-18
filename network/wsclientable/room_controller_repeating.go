package wsclientable

import (
	"log"
	"math"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/synchronization"
)

//Idea: See 'room_controller_http_editable.go'
//      Additionally we have two more fields in the http request headers:
//     		'first_time_unix_in_seconds_from_now' and 'repeat_every_seconds' and 'duration_in_seconds'
//          we use the 'seconds from now' semantic to minimize the problem of possibly differences in system time
//          	(it will be properly converted and stored in unix-time timestamps)
//              (note: since we assume localhost, this is likely not required - though nice to have)
//      A repeating room will only allow clients in the given time frame
//          e.g. clients can only connect within the given time and will be disconnected at the end of that timeframe
//          the calculation is the following: (c - f)/r <= l
// Example editing requests (python3):
//     import requests; r = requests.post("http://localhost:8089/rooms/repeat/add?id=test&allowed_clients=["c", "s", "parent"]&first_time_unix_in_seconds_from_now=10&repeat_every_seconds=10&duration_in_seconds=5"); print(r.reason, r.text)
//     import requests; r = requests.post("http://localhost:8089/rooms/repeat/edit?id=test&allowed_clients=["c", "s", "parent", "admin"]&first_time_unix_in_seconds_from_now=0&&repeat_every_seconds=10&duration_in_seconds=20"); print(r.reason, r.text)
//         NOTE: when editing the seconds_from_now is calculated again,
//                so it makes sense to have first_time_unix_in_seconds_from_now=0,
//                  otherwise some might be temporarily in an invalid room (leads to automatic disconnect)
//     import requests; r = requests.post("http://localhost:8089/rooms/repeat/remove?id=test"); print(r.reason, r.text)

type RepeatingRoomController struct {
	EditableRoomController
	nextExpiration synchronization.TimedCallback
}

func NewRepeatingRoomController(roomStorage RoomStorageI) *RepeatingRoomController {
	rc := &RepeatingRoomController{
		EditableRoomController: NewEditableRoomController(roomStorage),
		nextExpiration:         synchronization.NewTimedCallback(),
	}
	rc.reInitCallback()
	return rc
}

func (p *RepeatingRoomController) Close() error {
	p.nextExpiration.Stop()
	return p.EditableRoomController.Close()
}

func (p *RepeatingRoomController) NewConnectionForRoom(roomID string, connection ClientConnection) bool {
	room := p.GetRoom(roomID)
	_, userID, e := ConnectionIDStringToRoomIDAndUserID(connection.ID)
	if e == nil && room != nil && room.IsAllowed(userID) {
		rRoom := room.(RepeatingRoom)
		p.cleanAtAppropriateTimeForRepeatingRoom(&rRoom)
		return p.AddConnectionInRoomIfNotConnected(roomID, connection)
	}
	return false
}
func (p *RepeatingRoomController) CloseAndRemoveRoom(roomID string) (bool, error) {
	roomI := p.store.Get(roomID)
	if roomI == nil {
		return false, nil
	}
	oldRoom := roomI.(RepeatingRoom)
	if ExpirationCallbackDateForRepeatingRoom(&oldRoom) == p.nextExpiration.GetCallbackExpectedAt() {
		//damn... the closed room is the one that will be the next to expire, need to recompute
		p.reInitCallback()
	}

	_, e1 := p.CloseAllInRoom(roomID)
	existed, e2 := p.store.Remove(roomID)
	if e1 != nil {
		return existed, e1
	}
	return existed, e2
}
func (p *RepeatingRoomController) AddRoom(roomID string, newRoomI RoomI, allowOverride bool) error {
	newRoom := newRoomI.(RepeatingRoom)

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

	p.cleanAtAppropriateTimeForRepeatingRoom(&newRoom)
	return nil
}

func (p *RepeatingRoomController) cleanAtAppropriateTimeForRepeatingRoom(room *RepeatingRoom) {
	if room == nil {
		return
	}
	callbackAt := ExpirationCallbackDateForRepeatingRoom(room)
	wasEarlier := p.nextExpiration.CallMeBackIfEarlierThanCurrent(callbackAt, func() {
		p.reInitCallback()
	})
	log.Println("Callback for room(", room.ID, ") will be at the latest at", callbackAt, "(was earlier than current: ", wasEarlier, ")")
}
func (p *RepeatingRoomController) reInitCallback() {
	log.Println("Cleaning Expired Repeating Rooms")
	next := p.validateAllConnections()
	if next != nil {
		log.Printf("Next Room To Expire: %v, at %v", next.ID, ExpirationCallbackDateForRepeatingRoom(next))
	} else {
		log.Println("No Next Repeating Room To Expire (no repeating rooms with currently connected clients should exist)")
	}
	p.cleanAtAppropriateTimeForRepeatingRoom(next)
}

func ExpirationCallbackDateForRepeatingRoom(r *RepeatingRoom) time.Time {
	return time.Unix(ExpirationCallbackDateForRepeatingRoomWithUnix(r, time.Now().Unix()), 0).Add(time.Second) //a little after, so the callback is definitely after the expiration so the checks are successful
}
func ExpirationCallbackDateForRepeatingRoomWithUnix(r *RepeatingRoom, nowUnix int64) int64 {
	if r.DurationInSeconds >= r.RepeatEverySeconds {
		return math.MaxInt64 //then call back at the end of the universe
	}
	if nowUnix < r.FirstTimeUnixTimestamp {
		return r.FirstTimeUnixTimestamp + r.DurationInSeconds //then call back on first expiration
	}

	//log.Printf("CalcExpDateFor room: %v - fT=%v, now=%v, repeat=%v" +
	//	" - formula=((%v - (%v-(%v+%v)) mod %v) + %v)",
	//	r.ID, time.Unix(r.FirstTimeUnixTimestamp, 0), time.Unix(nowUnix, 0), r.RepeatEverySeconds,
	//	nowUnix, nowUnix, r.FirstTimeUnixTimestamp, r.DurationInSeconds, r.RepeatEverySeconds, r.RepeatEverySeconds)
	nowUnix -= 1 // this is done, so that if nowUnix == r.FirstTimeUnixTimestamp+r.DurationInSeconds, expiration = r.FirstTimeUnixTimestamp+r.DurationInSeconds
	// this is safe, because r.RepeatEverySeconds > r.DurationInSeconds
	nextExpirationAt := (nowUnix - Pmod(nowUnix-(r.FirstTimeUnixTimestamp+r.DurationInSeconds), r.RepeatEverySeconds)) + r.RepeatEverySeconds

	return nextExpirationAt
}

// Returns the next room to no longer be valid
func (p *RepeatingRoomController) validateAllConnections() *RepeatingRoom {
	var nextToBecomeInvalid *RepeatingRoom
	var currentExpirationDate time.Time
	p.ForAllRooms(func(roomID string) {
		r := p.validateConnectionsInRoom(roomID)
		expirationOfR := ExpirationCallbackDateForRepeatingRoom(r)
		if r.IsValid() && (nextToBecomeInvalid == nil || expirationOfR.Before(currentExpirationDate)) {
			currentExpirationDate = expirationOfR
			nextToBecomeInvalid = r
		}
	})
	return nextToBecomeInvalid
}
func (p *RepeatingRoomController) validateConnectionsInRoom(roomID string) *RepeatingRoom {
	room := p.store.Get(roomID).(RepeatingRoom)
	if !room.IsValid() {
		_, _ = p.CloseAndRemoveRoom(roomID)
	} else {
		p.ForAllIn(roomID, func(connection *ClientConnection) {
			_, userID, e := ConnectionIDStringToRoomIDAndUserID(connection.ID)
			if e != nil || !room.IsAllowed(userID) {
				e := connection.Close()
				if e != nil {
					log.Printf("IGNORED error on connection("+connection.ID+") close: %v", e)
				}
			}
		})
	}
	return &room
}
