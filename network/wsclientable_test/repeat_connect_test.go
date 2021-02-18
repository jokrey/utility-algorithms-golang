package wsclientable_test

import (
	"log"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestConnectAndSendTwice(t *testing.T) {
	go func() {
		sendTestMessage(t, time.Second*1, time.Second*1,
			"testRoom", "u1", "u2")
	}()

	go func() {
		sendTestMessage(t, time.Second*4, time.Second*1,
			"testRoom", "u1", "u2")
	}()

	go func() {
		time.Sleep(1 * time.Second)

		client, err := wsclientable.Connect("http://localhost:9087/room/test?room=testRoom&user=u2")
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

		if len(receivedMessagesSenderName) != 2 {
			t.Fatalf("Received wrong number of messages, senders: %v", receivedMessagesSenderName)
		}
	}()

	startPermanentRoomTestServer(10 * time.Second)
}

func sendTestMessage(t *testing.T, wait1, wait2 time.Duration, roomID, ownUserID, sendToUserID string) {
	time.Sleep(wait1)

	client, err := wsclientable.Connect("http://localhost:9087/room/test?room=" + roomID + "&user=" + ownUserID)
	if err != nil {
		t.Fatalf("could not start client, error: %v", err)
	}

	time.Sleep(wait2) // wait for receiver to connect

	err = client.SendTyped("roomForward", "{\"to\":\""+sendToUserID+"\"}")
	if err != nil {
		t.Fatalf("error sending data")
	}

	_ = client.Close()
}

func startPermanentRoomTestServer(maxTestDuration time.Duration) {
	base := wsclientable.NewWSHandlingServer()
	base.AddRoomForwardingFunctionality(
		wsclientable.BundleControllers(
			wsclientable.NewPermanentRoomController(wsclientable.NewPermanentRoom("testRoom", []string{"u1", "u2"})),
		), "roomForward",
	)

	go func() {
		log.Printf("Started RoomSignalingServer on %v:%v", "localhost", "9087")
		_ = base.StartUnencrypted("0.0.0.0", 9087, "/room/test")
	}()

	time.Sleep(maxTestDuration)
	_ = base.Close()
}
