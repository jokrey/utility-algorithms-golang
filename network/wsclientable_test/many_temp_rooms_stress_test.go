package wsclientable

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"log"
	"strconv"
	"testing"
	"time"
)

func TestManyTemporaryPersistentRooms(t *testing.T) {
	//test:
	//  server exists from [0, 20] and [25, 40]
	//  at 5s - room "best" is created
	//  room "best" exists from second 10 to second 30
	//  at 7s - attempt connect to best - expect fail
	//  at 15s - attempt connect to best - expect success
	//  at 27s - attempt connect to best - expect success
	//  at 32s - attempt connect to best - expect fail
	numOfRooms := 100

	startAt := time.Now()
	startAtUnix := time.Now().Unix()
	roomController := wsclientable.NewTemporaryRoomController(wsclientable.NewTemporaryRoomBoltStorage("test_temp_many.db"))
	for i := 0; i < numOfRooms; i++ {
		err := roomController.AddRoom("t"+strconv.Itoa(i),
			wsclientable.NewTemporaryRoom("t"+strconv.Itoa(i), []string{}, startAtUnix+10, startAtUnix+30), true)
		if err != nil {
			t.Fatalf("could not add room, error: %v", err)
		}
	}

	elapsedDuration := time.Now().Sub(startAt)

	if elapsedDuration > 7*time.Second {
		log.Print("took too long to create rooms to run all timed tests... - it is still possible to run some, which we will attempt - but consider using less rooms or buying a faster drive")
	}

	go func() {
		waitTime := 20 * time.Second
		if elapsedDuration > waitTime {
			log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
			return
		}

		log.Printf("server 1 started")
		startRoomTestServer(15207, waitTime-elapsedDuration, roomController)
		log.Printf("server 1 closed")
	}()

	for c := 0; c < numOfRooms; c++ {
		i := c

		go func() {
			waitTime := 7 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			_, err := wsclientable.Connect("http://localhost:15207/room/test?room=t" + strconv.Itoa(i) + "&user=user")
			if err == nil {
				t.Fatalf("could connect at %v, %v", waitTime, err)
			}
		}()

		go func() {
			waitTime := 15 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15207/room/test?room=t" + strconv.Itoa(i) + "&user=user")
			if err != nil {
				t.Fatalf("could not connect at %v, %v", waitTime, err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()

		go func() {
			waitTime := 26 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15207/room/test?room=t" + strconv.Itoa(i) + "&user=user")
			if err != nil {
				t.Fatalf("could not connect at %v, %v", waitTime, err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()

		go func() {
			waitTime := 35 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			_, err := wsclientable.Connect("http://localhost:15207/room/test?room=t" + strconv.Itoa(i) + "&user=user")
			if err == nil {
				t.Fatalf("could connect at %v, %v", waitTime, err)
			}
		}()
	}

	waitTime := 23 * time.Second
	if elapsedDuration > waitTime {
		log.Fatalf("Creating rooms took too long")
		return
	}
	time.Sleep(waitTime - elapsedDuration)

	roomController = wsclientable.NewTemporaryRoomController(wsclientable.NewTemporaryRoomBoltStorage("test_temp_many.db"))
	log.Printf("server 2 started")
	startRoomTestServer(15207, 20*time.Second, roomController)
	log.Printf("server 2 closed")
}
