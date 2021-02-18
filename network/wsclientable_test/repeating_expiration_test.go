package wsclientable

import (
	"log"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestRepeatingExpiration(t *testing.T) {
	startedAt := time.Now()
	go func() {
		time.Sleep(1 * time.Second)
		log.Printf("sender 1 woken up")

		client, err := wsclientable.Connect("http://localhost:15341/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=1, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server

		assertElapsedBetween(t, startedAt, 3, 6)
		log.Println("Client1 closed - elapsed:", time.Now().Sub(startedAt))
	}()

	go func() {
		time.Sleep(5 * time.Second)
		log.Printf("sender 2 woken up")

		_, err := wsclientable.Connect("http://localhost:15341/room/test?room=t&user=user1")
		if err == nil {
			t.Fatalf("could connect at s=5, %v", err)
		}
	}()
	go func() {
		time.Sleep(7 * time.Second)
		log.Printf("sender 3 woken up")

		client, err := wsclientable.Connect("http://localhost:15341/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=7, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server

		assertElapsedBetween(t, startedAt, 9, 12)
		log.Println("Client1 closed - elapsed:", time.Now().Sub(startedAt))
	}()

	roomController := wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat_expiration.db"))
	log.Println("Adding Room t manually without http")
	err := roomController.AddRoom("t", wsclientable.NewRepeatingRoom("t", []string{}, startedAt.Unix(), 6, 3), true)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("server started")
	startRoomTestServer(15341, 20*time.Second, roomController)
	log.Printf("server closed")
}

func assertElapsedBetween(t *testing.T, startedAt time.Time, afterSecs int, beforeSecs int) {
	now := time.Now()
	if !Between(now, startedAt.Add(time.Duration(afterSecs)*time.Second), startedAt.Add(time.Duration(beforeSecs)*time.Second)) {
		t.Fatalf("Client was not closed in expected timeframe, instead closed at: %v", now)
	}
}

func Between(t time.Time, tAfter time.Time, tBefore time.Time) bool {
	return t.After(tAfter) && t.Before(tBefore)
}
