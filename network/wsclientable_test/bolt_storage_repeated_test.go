package wsclientable

import (
	"os"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestBoltDbRepeatingRoom(t *testing.T) {
	now := time.Now().Unix()
	tR1Name := "isdfva - tR1"
	tR2Name := "ayxc23 - tR2"
	tR3Name := "yx113v47q5bov84eayxc23 - tR3"
	tR1 := wsclientable.RepeatingRoom{
		Room:                   wsclientable.Room{ID: tR1Name},
		FirstTimeUnixTimestamp: now,
		RepeatEverySeconds:     10000000,
		DurationInSeconds:      100,
	}
	tR2 := wsclientable.RepeatingRoom{
		Room:                   wsclientable.Room{ID: tR2Name},
		FirstTimeUnixTimestamp: now,
		RepeatEverySeconds:     6,
		DurationInSeconds:      3,
	}
	tR3 := wsclientable.RepeatingRoom{
		Room:                   wsclientable.Room{ID: tR3Name},
		FirstTimeUnixTimestamp: now + 3,
		RepeatEverySeconds:     10000000,
		DurationInSeconds:      100,
	}

	dbPath := "test_repeating_solo.db"
	err := os.RemoveAll(dbPath)
	if err != nil {
		t.Fatalf("could not remove db path: %v", err)
	}

	store := wsclientable.NewRepeatingRoomBoltStorage(dbPath)

	err = store.Put(tR1, false)
	if err != nil {
		t.Fatalf("could not put tR1: %v", err)
	}
	err = store.Put(tR2, false)
	if err != nil {
		t.Fatalf("could not put tR2: %v", err)
	}
	err = store.Put(tR3, false)
	if err != nil {
		t.Fatalf("could not put tR3: %v", err)
	}

	if !tR1.IsAllowed("egal") {
		t.Fatalf("tR1 not allowed at sec=0")
	}
	if !tR2.IsAllowed("egal") {
		t.Fatalf("tR2 not allowed at sec=0")
	}
	if tR3.IsAllowed("egal") {
		t.Fatalf("tR3 allowed at sec=0")
	}

	time.Sleep(time.Second * 4)

	if !tR1.IsAllowed("egal") {
		t.Fatalf("tR1 not allowed at sec=4")
	}
	if tR2.IsAllowed("egal") {
		t.Fatalf("tR2 allowed at sec=4")
	}
	if !tR3.IsAllowed("egal") {
		t.Fatalf("tR3 not allowed at sec=4")
	}

	time.Sleep(time.Second * 3)

	if !tR1.IsAllowed("egal") {
		t.Fatalf("tR1 not allowed at sec=7")
	}
	if !tR2.IsAllowed("egal") {
		t.Fatalf("tR2 not allowed at sec=7")
	}
	if !tR3.IsAllowed("egal") {
		t.Fatalf("tR3 not allowed at sec=7")
	}

	time.Sleep(time.Second * 3)

	if !tR1.IsAllowed("egal") {
		t.Fatalf("tR1 not allowed at sec=10")
	}
	if tR2.IsAllowed("egal") {
		t.Fatalf("tR2 allowed at sec=10")
	}
	if !tR3.IsAllowed("egal") {
		t.Fatalf("tR3 not allowed at sec=10")
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("could not close db: %v", err)
	}

	err = os.Remove(dbPath)
	if err != nil {
		t.Fatalf("could not remove db path: %v", err)
	}
}
