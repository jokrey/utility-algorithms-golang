package wsclientable_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestTemporaryRooms(t *testing.T) {
	//test:
	//  room 1 exists from second 3 to second 8

	base := wsclientable.NewWSHandlingServer()

	go func() {
		time.Sleep(1 * time.Second) // wait for server to start
		_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
		if err == nil {
			t.Fatalf("could connect before creation, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(6 * time.Second) // wait for other client to connect
		log.Printf("Attempt connect second time - while other still connected")
		_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
		if err == nil {
			t.Fatalf("could connect twice, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(9 * time.Second) // wait for other room to close
		log.Printf("Attempt connect after room dead")
		_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
		if err == nil {
			t.Fatalf("could connect after removal, error: %v", err)
		}
	}()
	go func() {
		time.Sleep(4 * time.Second) // wait for other client to connect
		client, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
		if err != nil {
			t.Fatalf("error sending data")
		}

		// if the following completes, we know that the server kicked this client (because room closed by room editor)
		client.ListenLoop(wsclientable.MessageHandlers{})
		_ = base.Close()
	}()
	go func() {
		time.Sleep(4 * time.Second) // wait for other client to connect
		client, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u2")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		var receivedMessagesSenderName []string
		msgHandlers := map[string]func(string, wsclientable.ClientConnection, map[string]interface{}){
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
		response, err := http.Get("http://localhost:10013/add?id=tempRoom1&allowed_clients=[]&valid_from_in_seconds_from_now=2&valid_until_in_seconds_from_now=6")
		if err != nil {
			t.Fatalf("could not add room: %v", err)
		}

		fmt.Printf("Response on add: %v", response)
	}()

	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewPermanentRoomController(
				wsclientable.NewPermanentRoom("testRoom", []string{"u1", "u2"}),
				wsclientable.NewPermissiblePermanentRoom("testRoom2"),
			),
			wsclientable.NewHTTPRoomEditorInRam("localhost", 10012, "/add", "/edit", "/remove"),
			wsclientable.NewHTTPTemporaryRoomEditorInRam(
				"localhost", 10013,
				"/add", "/edit", "/remove"),
		),
		"roomForward",
	)

	log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "10011")
	_ = base.StartUnencrypted("0.0.0.0", 10011, "/room/test")
}
