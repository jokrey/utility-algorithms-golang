package wsclientable_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestTemporaryPersistentRooms(t *testing.T) {
	//test:
	//  room 1 exists from second 3 to second 8

	base := wsclientable.NewWSHandlingServer()

	go func() {
		time.Sleep(1 * time.Second) // wait for other client to connect
		_, err := wsclientable.Connect("http://localhost:20011/room/test?room=tempRoom1&user=u1")
		if err == nil {
			t.Fatalf("could connect before creation, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(6 * time.Second) // wait for other client to connect
		_, err := wsclientable.Connect("http://localhost:20011/room/test?room=tempRoom1&user=u1")
		if err == nil {
			t.Fatalf("could connect twice, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(11 * time.Second) // wait for other client to connect
		_, err := wsclientable.Connect("http://localhost:20011/room/test?room=tempRoom1&user=uNEW")
		if err == nil {
			_ = base.Close()
			t.Fatalf("could connect after removal, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(5 * time.Second) // wait for other client to connect
		client, err := wsclientable.Connect("http://localhost:20011/room/test?room=tempRoom1&user=u1")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
		if err != nil {
			t.Fatalf("error sending data")
		}

		// if the following does not complete, we know that the server only closes, if it kicks this client
		client.ListenLoop(wsclientable.MessageHandlers{})
		_ = base.Close()
	}()
	go func() {
		time.Sleep(4 * time.Second) // wait for other client to connect
		client, err := wsclientable.Connect("http://localhost:20011/room/test?room=tempRoom1&user=u2")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		var receivedMessagesSenderName []string
		msgHandlers := wsclientable.MessageHandlers{
			"roomForward": func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
				receivedMessagesSenderName = append(receivedMessagesSenderName, data["from"].(string))
			},
		}

		client.ListenLoop(msgHandlers)

		if len(receivedMessagesSenderName) != 1 {
			t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
		}
	}()
	go func() {
		time.Sleep(1 * time.Second)

		println("starting to send")
		response, err := http.Get("http://localhost:20014/edit?id=tempRoom1&allowed_clients=[]&valid_from_in_seconds_from_now=2&valid_until_in_seconds_from_now=8")
		if err != nil {
			t.Fatalf("could not edit room: %v", err)
		}

		fmt.Printf("Response on edit: %v", response)
	}()

	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewHTTPTemporaryPersistedRoomEditor(
				"localhost", 20014,
				"/add", "/edit", "/remove",
				"test_rooms.db"),
		),
		"roomForward",
	)

	log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "20011")
	_ = base.StartUnencrypted("0.0.0.0", 20011, "/room/test")
}
