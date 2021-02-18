package wsclientable

import (
	"github.com/jokrey/utility-algorithms-golang/network/wsclientable"
	"log"
	"strconv"
	"testing"
	"time"
)

func TestManyRepeatingRooms(t *testing.T) {
	numOfRooms := 200

	startAt := time.Now()
	startAtUnix := time.Now().Unix()
	roomController := wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat_many.db"))
	for i := 0; i < numOfRooms; i++ {
		err := roomController.AddRoom("t"+strconv.Itoa(i),
			wsclientable.NewRepeatingRoom("t"+strconv.Itoa(i), []string{}, startAtUnix, 10, 5), true)
		if err != nil {
			t.Fatalf("could not add room, error: %v", err)
		}
	}

	elapsedDuration := time.Now().Sub(startAt)

	if elapsedDuration > 2*time.Second {
		log.Print("took too long to create rooms to run all timed tests... - it is still possible to run some, which we will attempt - but consider using less rooms or buying a faster drive")
	}

	go func() {
		waitTime := 20 * time.Second
		if elapsedDuration > waitTime {
			log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
			return
		}

		log.Printf("server 1 started")
		startRoomTestServer(15205, waitTime-elapsedDuration, roomController)
		log.Printf("server 1 closed")
	}()

	for c := 0; c < numOfRooms; c++ {
		i := c
		go func() {
			waitTime := 2 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user1")
			if err != nil {
				t.Fatalf("could not connect at s=2, %v", err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()

		go func() {
			waitTime := 7 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			_, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user2")
			if err == nil {
				t.Fatalf("could connect at s=7")
			}
		}()

		go func() {
			waitTime := 12 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user1")
			if err != nil {
				t.Fatalf("could not connect at s=12, %v", err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()

		go func() {
			waitTime := 27 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			_, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user4")
			if err == nil {
				t.Fatalf("could connect at s=27")
			}
		}()

		go func() {
			waitTime := 32 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user1")
			if err != nil {
				t.Fatalf("could not connect at s=32, %v", err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()

		go func() {
			waitTime := 37 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			_, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user6")
			if err == nil {
				t.Fatalf("could connect at s=37")
			}
		}()

		go func() {
			waitTime := 43 * time.Second
			if elapsedDuration > waitTime {
				log.Printf("HAD TO SKIP TEST WITH WAIT_TIME = %v, because already elapsedDuration(%v)", waitTime, elapsedDuration)
				return
			}
			time.Sleep(waitTime - elapsedDuration)

			client, err := wsclientable.Connect("http://localhost:15205/room/test?room=t" + strconv.Itoa(i) + "&user=user1")
			if err != nil {
				t.Fatalf("could not connect at s=43, %v", err)
			}
			client.ListenLoop(wsclientable.MessageHandlers{}) //wait until closed by server
		}()
	}

	waitTime := 25 * time.Second
	if elapsedDuration > waitTime {
		log.Fatalf("Creating rooms took too long")
		return
	}
	time.Sleep(waitTime - elapsedDuration)

	roomController = wsclientable.NewRepeatingRoomController(wsclientable.NewRepeatingRoomBoltStorage("test_repeat_many.db"))
	log.Printf("server 2 started")
	startRoomTestServer(15205, 25*time.Second, roomController)
	log.Printf("server 2 closed")
}
