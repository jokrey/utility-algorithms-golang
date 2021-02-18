package synchronization_test

import (
	"github.com/jokrey/utility-algorithms-golang/network/synchronization"
	"log"
	"testing"
	"time"
)

func Test(t *testing.T) {
	tcb := synchronization.NewTimedCallback()
	successCounter := 0

	tcb.CallMeBackAfter(time.Second, func() {
		log.Printf("after 1 second")
		successCounter++
	})
	tcb.CallMeBackIfEarlierThanCurrent(time.Now().Add(4*time.Second), func() {
		log.Printf("should not have been called (4)")
		t.Fail()
	})

	time.Sleep(2 * time.Second)

	tcb.CallMeBackAt(time.Now().Add(4*time.Second), func() {
		log.Printf("should not have been called (2+4)")
		t.Fail()
	})
	tcb.CallMeBackIfEarlierThanCurrent(time.Now().Add(2*time.Second), func() {
		log.Printf("after 3 second")
		successCounter++
	})

	time.Sleep(5 * time.Second)

	if successCounter != 2 {
		log.Fatalf("success != expected(2)")
	}
}

func TestRecursive(t *testing.T) {
	tcb := synchronization.NewTimedCallback()

	counter := 0
	var recursive func()
	recursive = func() {
		tcb.CallMeBackAfter(time.Second*1, func() {
			recursive()
			counter++
		})
	}
	recursive()

	time.Sleep(5 * time.Second)
	if counter < 4 {
		t.Fatalf("counter only = %v - apparently not called recursively", counter)
	}
}
