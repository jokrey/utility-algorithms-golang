package wsclientable

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestAdvancedTemporaryPersistentRooms(t *testing.T) {
	//test:
	//  server exists from [0, 20] and [25, 40]
	//  at 5s - room "best" is created
	//  room "best" exists from second 10 to second 30
	//  at 7s - attempt connect to best - expect fail
	//  at 15s - attempt connect to best - expect success
	//  at 27s - attempt connect to best - expect success
	//  at 32s - attempt connect to best - expect fail

	go func() {
		time.Sleep(7 * time.Second)
		_, err := wsclientable.Connect("http://localhost:10211/room/test?room=best&user=user1")
		if err == nil {
			t.Fatalf("Could connect at t=7s")
		}
	}()
	go func() {
		time.Sleep(15 * time.Second)
		client, err := wsclientable.Connect("http://localhost:10211/room/test?room=best&user=user2")
		if err != nil {
			t.Fatalf("Could NOT connect at t=15s")
		}
		_ = client.Close()
	}()
	go func() {
		time.Sleep(27 * time.Second)
		client, err := wsclientable.Connect("http://localhost:10211/room/test?room=best&user=user3")
		if err != nil {
			t.Fatalf("Could NOT connect at t=27s")
		}
		_ = client.Close()
	}()
	go func() {
		time.Sleep(32 * time.Second)
		_, err := wsclientable.Connect("http://localhost:10211/room/test?room=best&user=user4")
		if err == nil {
			t.Fatalf("Could connect at t=32s")
		}
	}()

	go func() {
		base := wsclientable.NewWSHandlingServer()
		//t.Run("server-runner 1", func(t *testing.T) {
		//	t.Parallel()
		//	log.Printf("server started at 0s")
		//	restartPersTempRoomForwardingServer(t, base)
		//})
		go func() {
			log.Printf("server started at 0s")
			restartPersTempRoomForwardingServer(&base)
		}()
		time.Sleep(20 * time.Second)
		log.Printf("closing server 0-20")
		err := base.Close()
		if err != nil {
			log.Fatalf("error closing server(1): %v", err)
		}
	}()
	go func() {
		time.Sleep(5 * time.Second)

		addRoom(t, "best", "5", "25")
	}()

	time.Sleep(25 * time.Second)

	base := wsclientable.NewWSHandlingServer()
	go func() {
		log.Printf("server started at 25s")
		restartPersTempRoomForwardingServer(&base)
	}()
	time.Sleep(15 * time.Second)
	log.Printf("closing server 25-40")
	err := base.Close()
	if err != nil {
		log.Fatalf("error closing server(2): %v", err)
	}
}

func addRoom(t *testing.T, roomID, fromSeconds, untilSeconds string) {
	response, err := http.Get("http://localhost:10214/edit?id=" + roomID + "&allowed_clients=[]&valid_from_in_seconds_from_now=" + fromSeconds + "&valid_until_in_seconds_from_now=" + untilSeconds)
	if err != nil {
		t.Fatalf("Error editing room %v", err)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	log.Printf(bodyString)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Error editing room %v", response)
	}
}

func restartPersTempRoomForwardingServer(base *wsclientable.Server) {
	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewHTTPTemporaryPersistedRoomEditor(
				"localhost", 10214,
				"/add", "/edit", "/remove",
				"test_rooms_adv2.db"),
		),
		"roomForward",
	)

	log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "10211")
	_ = base.StartUnencrypted("0.0.0.0", 10211, "/room/test")
}
