package wsclientable

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"log"
	"testing"
	"time"
)

func TestRepeatingRooms(t *testing.T) {
	go func() {
		roomController := wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat_adv.db"))
		go func() {
			err := roomController.AddRoom("t", wsclientable.NewRepeatingRoom("t", []string{}, time.Now().Unix(), 10, 5), true)
			if err != nil {
				t.Fatalf("could not start client, error: %v", err)
			}
		}()
		log.Printf("server 1 started")
		startRoomTestServer(15201, 20*time.Second, roomController)
		log.Printf("server 1 closed")
	}()

	go func() {
		time.Sleep(2 * time.Second)
		log.Printf("sender 1 woken up")

		client, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=2, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
	}()

	go func() {
		time.Sleep(7 * time.Second)
		log.Printf("sender 2 woken up")

		_, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user2")
		if err == nil {
			t.Fatalf("could connect at s=7")
		}
	}()

	go func() {
		time.Sleep(12 * time.Second)
		log.Printf("sender 3 woken up")

		client, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=12, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
	}()

	go func() {
		time.Sleep(27 * time.Second)
		log.Printf("sender 4 woken up")

		_, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user4")
		if err == nil {
			t.Fatalf("could connect at s=27")
		}
	}()

	go func() {
		time.Sleep(32 * time.Second)
		log.Printf("sender 5 woken up")

		client, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=32, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
	}()

	go func() {
		time.Sleep(37 * time.Second)
		log.Printf("sender 6 woken up")

		_, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user6")
		if err == nil {
			t.Fatalf("could connect at s=37")
		}
	}()

	go func() {
		time.Sleep(43 * time.Second)
		log.Printf("sender 7 woken up")

		client, err := wsclientable.Connect("http://localhost:15201/room/test?room=t&user=user1")
		if err != nil {
			t.Fatalf("could not connect at s=43, %v", err)
		}
		client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
	}()

	time.Sleep(25 * time.Second) //wait until other server

	roomController := wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat_adv.db"))
	log.Printf("server 2 started")
	startRoomTestServer(15201, 25*time.Second, roomController)
	log.Printf("server 2 closed")
}

func startRoomTestServer(port int, maxTestDuration time.Duration, roomController wsclientable.RoomControllerI) {
	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			roomController,
		), "roomForward",
	)

	go func() {
		log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "15201")
		_ = base.StartUnencrypted("0.0.0.0", port, "/room/test")
	}()

	time.Sleep(maxTestDuration)
	_ = base.Close()
}
