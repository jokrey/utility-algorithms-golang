package wsclientable_test

import (
	"fmt"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestPermanentRooms(t *testing.T) {
	t.Parallel()
	t.Run("rooms-test", func(t *testing.T) {
		t.Parallel()
		t.Run("Server", func(t *testing.T) {
			t.Parallel()
			startRoomTestServer(3 * time.Second)
		})
		t.Run("InvalidRoomClient-Sender", func(t *testing.T) {
			t.Parallel()

			_, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoomWRONGasdasdasd&user=u3")
			if err == nil { // we EXPECT an error here, since the room does not exist
				t.Fatalf("error authentication success on invalid room")
			}
		})
		t.Run("SameRoomWrongClient-Sender", func(t *testing.T) {
			t.Parallel()

			_, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom&user=u3")
			if err == nil { // we EXPECT an error here, since the room does not exist
				t.Fatalf("error authentication success on invalid room")
			}
		})
		t.Run("SameRoomClient-Sender", func(t *testing.T) {
			t.Parallel()

			client, err := wsclientable.Connect("http://localhost:9011/room/test?room=testRoom&user=u1")
			if err != nil {
				t.Fatalf("could not start client, error: %v", err)
			}

			time.Sleep(1 * time.Second) // wait for other client to connect

			err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
			if err != nil {
				t.Fatalf("error sending data")
			}

			_ = client.Close()
		})
		t.Run("SameRoomClient-Receiver", func(t *testing.T) {
			t.Parallel()

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
			client.ListenLoop(msgHandlers, []func(wsclientable.ClientConnection, int, string){})

			if len(receivedMessagesSenderName) != 1 {
				t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
			}
		})

		t.Run("SameRoom2Client-Sender", func(t *testing.T) {
			t.Parallel()

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
		})
		t.Run("SameRoom2Client-Receiver", func(t *testing.T) {
			t.Parallel()

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
			client.ListenLoop(msgHandlers, []func(wsclientable.ClientConnection, int, string){})

			if len(receivedMessagesSenderName) != 1 {
				t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
			}
		})
	})
}

func startRoomTestServer(maxTestDuration time.Duration) {
	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewPermanentRoomController(
				wsclientable.NewPermanentRoom("testRoom", []string{"u1", "u2"}),
				wsclientable.NewPermissiblePermanentRoom("testRoom2"),
			),
			wsclientable.NewHTTPEditableRoomController("localhost", 9012, "/add", "/edit", "/remove"),
			wsclientable.NewTemporaryRoomController("localhost", 9013, "/add", "/edit", "/remove"),
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

func TestTemporaryRooms(t *testing.T) {
	t.Parallel()

	//test:
	//  room 1 exists from second 3 to second 8

	base := wsclientable.NewWSHandlingServer()

	t.Run("rooms-test", func(t *testing.T) {
		t.Parallel()
		t.Run("CannotConnectYet-Sender", func(t *testing.T) {
			t.Parallel()

			time.Sleep(1 * time.Second) // wait for other client to connect
			_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
			if err == nil {
				t.Fatalf("could connect before creation, error: %v", err)
			}
		})
		t.Run("CannotConnectAgain-Sender", func(t *testing.T) {
			t.Parallel()

			time.Sleep(5 * time.Second) // wait for other client to connect
			_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
			if err == nil {
				t.Fatalf("could connect twice, error: %v", err)
			}
		})
		t.Run("CannotConnectAnymore-Sender", func(t *testing.T) {
			t.Parallel()

			time.Sleep(9 * time.Second) // wait for other client to connect
			_, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
			if err == nil {
				t.Fatalf("could connect after removal, error: %v", err)
			}
		})

		t.Run("SameRoomClient-Sender", func(t *testing.T) {
			t.Parallel()

			time.Sleep(4 * time.Second) // wait for other client to connect
			client, err := wsclientable.Connect("http://localhost:10011/room/test?room=tempRoom1&user=u1")
			if err != nil {
				t.Fatalf("could not start client, error: %v", err)
			}

			err = client.SendTyped("roomForward", "{\"to\":\"u2\"}")
			if err != nil {
				t.Fatalf("error sending data")
			}

			// if the following does not complete, we know that the server only closes, if it kicks this client
			client.ListenLoop(
				map[string]func(string, wsclientable.ClientConnection, map[string]interface{}){},
				[]func(wsclientable.ClientConnection, int, string){})
			_ = base.Close()
		})
		t.Run("SameRoomClient-Receiver", func(t *testing.T) {
			t.Parallel()

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

			client.ListenLoop(msgHandlers, []func(wsclientable.ClientConnection, int, string){})

			if len(receivedMessagesSenderName) != 1 {
				t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
			}
		})

		t.Run("Server", func(t *testing.T) {
			t.Parallel()

			base.AddRoomForwardingFunctionality(
				wsclientable.BundleControllers(
					wsclientable.NewPermanentRoomController(
						wsclientable.NewPermanentRoom("testRoom", []string{"u1", "u2"}),
						wsclientable.NewPermissiblePermanentRoom("testRoom2"),
					),
					wsclientable.NewHTTPEditableRoomController("localhost", 10012, "/add", "/edit", "/remove"),
					wsclientable.NewTemporaryRoomController("localhost", 10013, "/add", "/edit", "/remove"),
				),
				"roomForward",
			)

			log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "10011")
			_ = base.StartUnencrypted("0.0.0.0", 10011, "/room/test")
		})
		t.Run("RoomEditor", func(t *testing.T) {
			t.Parallel()
			time.Sleep(1 * time.Second)

			println("starting to send")
			response, err := http.Get("http://localhost:10013/add?id=tempRoom1&allowed_clients=[]&valid_from_in_seconds_from_now=2&valid_until_in_seconds_from_now=8")
			if err != nil {
				t.Fatalf("could not edit room: %v", err)
			}

			fmt.Printf("Response on edit: %v", response)
		})
	})
}
