package wsclientable_test

import (
	"log"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestPermanentRooms(t *testing.T) {
	go func() {
		time.Sleep(1 * time.Second)

		_, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoomWRONGasdasdasd&user=u3")
		if err == nil { // we EXPECT an error here, since the room does not exist
			t.Fatalf("error authentication success on invalid room")
		}
	}()
	go func() {
		time.Sleep(1 * time.Second)

		_, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom&user=u3")
		if err == nil { // we EXPECT an error here, since the room does not exist
			t.Fatalf("error authentication success on invalid room")
		}
	}()
	go func() {
		time.Sleep(1 * time.Second)

		client, err := wsclientable.ConnectToRoom("http://localhost:9011/room/test", "testRoom", "u1")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		time.Sleep(1 * time.Second) // wait for other client to connect

		err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
		if err != nil {
			t.Fatalf("error sending data")
		}

		_ = client.Close()
	}()
	go func() {
		time.Sleep(1 * time.Second)

		client, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom&user=u2")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		var receivedMessagesSenderName []string
		msgHandlers := map[string]func(string, wsclientable.ClientConnection, map[string]interface{}){
			"roomForward": func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
				receivedMessagesSenderName = append(receivedMessagesSenderName, data["from"].(string))
			},
		}

		// listener loop closed after 4 seconds by - server, eval how many messages where received
		client.ListenLoop(msgHandlers)

		if len(receivedMessagesSenderName) != 1 {
			t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
		}
	}()
	go func() {
		time.Sleep(1 * time.Second)

		client, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom2&user=u1")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		time.Sleep(1 * time.Second) // wait for other client to connect

		err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
		if err != nil {
			t.Fatalf("error sending data")
		}

		_ = client.Close()
	}()
	go func() {
		time.Sleep(1 * time.Second)

		client, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom2&user=u2")
		if err != nil {
			t.Fatalf("could not start client, error: %v", err)
		}

		var receivedMessagesSenderName []string
		msgHandlers := map[string]func(string, wsclientable.ClientConnection, map[string]interface{}){
			"roomForward": func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
				receivedMessagesSenderName = append(receivedMessagesSenderName, data["from"].(string))
			},
		}

		// listener loop closed after 4 seconds by - server, eval how many messages where received
		client.ListenLoop(msgHandlers)

		if len(receivedMessagesSenderName) != 1 {
			t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
		}
	}()

	startRoomTestServer(5 * time.Second)
}

func startRoomTestServer(maxTestDuration time.Duration) {
	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewPermanentRoomController(
				wsclientable.NewPermanentRoom("testRoom", []string{"u1", "u2"}),
				wsclientable.NewPermissiblePermanentRoom("testRoom2"),
			),
			wsclientable.NewHTTPRoomEditorInRam("localhost", 9012, "/add", "/edit", "/remove"),
			wsclientable.NewHTTPTemporaryRoomEditorInRam("localhost", 9013, "/add", "/edit", "/remove"),
		),
		"roomForward",
	)

	go func() {
		log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "9011")
		_ = base.StartUnencrypted("0.0.0.0", 9011, "/room/test")
	}()

	time.Sleep(maxTestDuration)
	_ = base.Close()
}
