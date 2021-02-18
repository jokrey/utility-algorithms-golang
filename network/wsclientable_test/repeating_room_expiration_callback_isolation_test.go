package wsclientable

import (
	"math"
	"testing"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestRepeatingRoomExpirationCallbackInIsolation(t *testing.T) {
	rr1 := wsclientable.NewRepeatingRoom("1", []string{}, 100, 10, 5)

	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 90) != 105 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 100) != 105 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 101) != 105 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 105) != 105 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 106) != 115 {
		t.Fatal("")
	}

	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 110) != 115 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 111) != 115 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 115) != 115 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr1, 116) != 125 {
		t.Fatal("")
	}

	rr2 := wsclientable.NewRepeatingRoom("2", []string{}, 100, 10, 10)
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr2, 90) != math.MaxInt64 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr2, 101) != math.MaxInt64 {
		t.Fatal("")
	}

	rr3 := wsclientable.NewRepeatingRoom("2", []string{}, 100, 7, 6)

	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 90) != 106 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 100) != 106 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 101) != 106 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 106) != 106 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 107) != 113 {
		t.Fatal("")
	}

	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 110) != 113 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 112) != 113 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 113) != 113 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 114) != 120 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 115) != 120 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 116) != 120 {
		t.Fatal("")
	}
	if wsclientable.ExpirationCallbackDateForRepeatingRoomWithUnix(&rr3, 120) != 120 {
		t.Fatal("")
	}

}
