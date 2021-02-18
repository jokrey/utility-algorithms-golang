package wsclientable

import (
	"log"
	"strconv"
	"testing"
	"time"
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
)

func TestRepeatingRoomsStress(t *testing.T) {
	startAt := time.Now()

	go func() {
		roomController := wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat.db"))
		go func() {
			err := roomController.AddRoom("t", wsclientable.NewRepeatingRoom("t", []string{}, time.Now().Unix(), 100, 10), true)
			if err != nil {
				t.Fatalf("could not start client, error: %v", err)
			}
		}()
		log.Printf("server 1 started")
		startRoomTestServer(15202, 60*time.Second, roomController)
		log.Printf("server 1 closed")
	}()

	numSendClients := 1000
	totalClientsDone := 0
	successReceiveCounter := 0

	time.Sleep(1 * time.Second)
	for i := 0; i < numSendClients; i++ {
		i := i
		go func() {
			client, err := wsclientable.Connect("http://localhost:15202/room/test?room=t&user=receiving-user" + strconv.Itoa(i))
			if err != nil {
				totalClientsDone++
				t.Fatalf("could not start client, error: %v", err)
			}

			var receivedMessagesSenderName []string
			msgHandlers := map[string]func(string, wsclientable.ClientConnection, map[string]interface{}){
				"roomForward": func(_ string, _ wsclientable.ClientConnection, data map[string]interface{}) {
					receivedMessagesSenderName = append(receivedMessagesSenderName, data["from"].(string))
				},
			}
			client.ListenLoop(msgHandlers)
			totalClientsDone++

			if len(receivedMessagesSenderName) != 1 {
				t.Fatalf(strconv.Itoa(i)+" received wrong number of messages, senders: %v", receivedMessagesSenderName)
			}
			successReceiveCounter++
		}()
	}

	time.Sleep(1 * time.Second)
	for i := 0; i < numSendClients; i++ {
		i := i
		go func() {
			client, err := wsclientable.Connect("http://localhost:15202/room/test?room=t&user=sending-user" + strconv.Itoa(i))
			if err != nil {
				totalClientsDone++
				t.Fatalf("could not start client("+strconv.Itoa(i)+"), error: %v", err)
			}

			err = client.SendTyped("roomForward", "{\"to\":\"receiving-user"+strconv.Itoa(i)+"\", \"data\":\"hallo\"}")
			totalClientsDone++
			if err != nil {
				t.Fatalf("could not send on client("+strconv.Itoa(i)+"), error: %v", err)
			}
		}()
	}

	stoppedAt := time.Now()
	for totalClientsDone < numSendClients*2 {
		time.Sleep(1 * time.Second)
	}

	timeTaken := stoppedAt.Sub(startAt)
	log.Printf("Time Taken: %v", timeTaken)

	if timeTaken >= 40*time.Second {
		t.Fatal("Took to long") //in this case the clients did not receive, but where kicked by repeating room closure
	}
	for successReceiveCounter < numSendClients {
		t.Fatal("Not all receivers received")
	}
}
