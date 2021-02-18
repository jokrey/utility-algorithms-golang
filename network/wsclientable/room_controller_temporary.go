package wsclientable

import (
	"log"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/synchronization"
)

//Idea: See 'room_controller_http_editable.go'
//      Additionally we have two more fields in the http request headers:
//     		'valid_from_in_seconds_from_now' and 'valid_until_in_seconds_from_now'
//          where 'valid_until_in_seconds_from_now' > 'valid_from_in_seconds_from_now' must hold
//          we use the 'seconds from now' semantic to minimize the problem of possibly differences in system time
//          	(it will be properly converted and stored in unix-time timestamps)
//              (note: since we assume localhost, this is likely not required - though nice to have)
//      A temporary room will only show as 'existing' for the given time frame.
//          e.g. clients can only connect within the given time and will be disconnected at the end of that timeframe
//  example editing requests (python3) - note: the concrete required call depends on the config file:
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/add?id=test&allowed_clients=["c", "s", "parent"]&valid_from_in_seconds_from_now=10&valid_until_in_seconds_from_now=1000"); print(r.reason, r.text)
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/edit?id=test&allowed_clients=["c", "s", "parent", "admin"]&valid_from_in_seconds_from_now=0&valid_until_in_seconds_from_now=20"); print(r.reason, r.text)
//         NOTE: when editing the seconds_from_now is calculated again,
//                so it makes sense to have valid_from_in_seconds_from_now=0,
//                  otherwise some might be temporarily in an invalid room (leads to automatic disconnect)
//     import requests; r = requests.post("http://localhost:8089/rooms/temp/control/remove?id=test"); print(r.reason, r.text)

type TemporaryRoomController struct {
	EditableRoomController
	nextExpiration synchronization.TimedCallback
}

func NewTemporaryRoomController(roomStorage RoomStorageI) *TemporaryRoomController {
	tc := &TemporaryRoomController{
		EditableRoomController: NewEditableRoomController(roomStorage),
		nextExpiration:         synchronization.NewTimedCallback(),
	}
	tc.reInitCallback()
	return tc
}

func (p *TemporaryRoomController) CloseAndRemoveRoom(roomID string) (bool, error) {
	roomI := p.store.Get(roomID)
	if roomI == nil {
		return false, nil
	}
	oldRoom := roomI.(TemporaryRoom)
	if expirationCallbackDateForTemporaryRoom(&oldRoom) == p.nextExpiration.GetCallbackExpectedAt() {
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

func (p *TemporaryRoomController) AddRoom(roomID string, newRoomI RoomI, allowOverride bool) error {
	newRoom := newRoomI.(TemporaryRoom)

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
	if newRoom.ValidFromUnixTime > 0 { // all current clients now in invalid room
		_, _ = p.CloseAllInRoom(roomID) //automatically removes connection in room also
		// do not delete - room will become active later
	}

	p.cleanAtAppropriateTimeForTemporaryRoom(&newRoom)
	return nil
}

func (p *TemporaryRoomController) cleanAtAppropriateTimeForTemporaryRoom(room *TemporaryRoom) {
	if room == nil {
		return
	}
	p.nextExpiration.CallMeBackIfEarlierThanCurrent(expirationCallbackDateForTemporaryRoom(room), func() {
		next, _ := p.store.(TemporaryRoomStorageI).CleanExpired(func(removed *TemporaryRoom) {
			_, _ = p.CloseAllInRoom(removed.GetID())
			log.Println("Room " + removed.GetID() + " expired and was removed")
		})
		p.cleanAtAppropriateTimeForTemporaryRoom(next)
	})
}
func expirationCallbackDateForTemporaryRoom(room *TemporaryRoom) time.Time {
	return time.Unix(room.ValidUntilUnixTime, 0).Add(time.Second) //a little after, so the callback is definitely after the expiration so the checks are successful
}

func (p *TemporaryRoomController) reInitCallback() {
	next, _ := p.store.(TemporaryRoomStorageI).CleanExpired(func(removed *TemporaryRoom) {
		_, _ = p.CloseAllInRoom(removed.GetID())
		log.Println("Room " + removed.GetID() + " expired and was removed")
	})
	p.cleanAtAppropriateTimeForTemporaryRoom(next)
}
