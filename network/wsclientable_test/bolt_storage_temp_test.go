package wsclientable

import (
	"os"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestBoltDbTemporaryRoom(t *testing.T) {
	now := time.Now().Unix()
	tR1Name := "isdfva - tR1"
	tR2Name := "ayxc23 - tR2"
	tR22Name := "3v47q5bov84eayxc23 - tR22"
	tR3Name := "yx113v47q5bov84eayxc23 - tR3"
	tR1 := wsclientable.TemporaryRoom{
		Room:               wsclientable.Room{ID: tR1Name},
		ValidFromUnixTime:  now,
		ValidUntilUnixTime: now + 5,
	}
	tR2 := wsclientable.TemporaryRoom{
		Room:               wsclientable.Room{ID: tR2Name},
		ValidFromUnixTime:  now,
		ValidUntilUnixTime: now + 3,
	}
	tR22 := wsclientable.TemporaryRoom{
		Room:               wsclientable.Room{ID: tR22Name},
		ValidFromUnixTime:  now,
		ValidUntilUnixTime: now + 3,
	}
	tR3 := wsclientable.TemporaryRoom{
		Room:               wsclientable.Room{ID: tR3Name},
		ValidFromUnixTime:  now,
		ValidUntilUnixTime: now + 2,
	}

	dbPath := "test_solo.db"
	err := os.RemoveAll(dbPath)
	if err != nil {
		t.Fatalf("could not remove db path: %v", err)
	}

	store := wsclientable.NewTemporaryRoomBoltStorage(dbPath)

	err = store.Put(tR1, false)
	if err != nil {
		t.Fatalf("could not put tR1: %v", err)
	}
	err = store.Put(tR2, false)
	if err != nil {
		t.Fatalf("could not put tR2: %v", err)
	}
	err = store.Put(tR22, false)
	if err != nil {
		t.Fatalf("could not put tR22: %v", err)
	}
	err = store.Put(tR3, false)
	if err != nil {
		t.Fatalf("could not put tR3: %v", err)
	}

	_, err = store.CleanExpired(func(room *wsclientable.TemporaryRoom) {
		t.Fatalf("Removed %v at second=0", room)
	})
	if err != nil {
		t.Fatalf("could not CleanExpired(1): %v", err)
	}

	time.Sleep(4 * time.Second)

	_, err = store.CleanExpired(func(room *wsclientable.TemporaryRoom) {})
	if err != nil {
		t.Fatalf("could not CleanExpired(2): %v", err)
	}
	if store.Get(tR1Name) == nil {
		t.Fatalf("Removed tr1 at second = 3")
	}
	if store.Get(tR3Name) != nil {
		t.Fatalf("Failed to remove tr3 at second = 3")
	}
	if store.Get(tR2Name) != nil {
		t.Fatalf("Failed to remove tr2 at second = 3")
	}
	if store.Get(tR22Name) != nil {
		t.Fatalf("Failed to remove tr22 at second = 3")
	}

	time.Sleep(2 * time.Second)
	_, err = store.CleanExpired(func(room *wsclientable.TemporaryRoom) {})
	if err != nil {
		t.Fatalf("could not CleanExpired(3): %v", err)
	}
	if store.Get(tR1Name) != nil {
		t.Fatalf("Failed to remove tr1 at second = 5")
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
