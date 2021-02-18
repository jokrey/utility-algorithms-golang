package synchronization

import (
	"sync"
	"time"
)

type TimedCallback struct {
	// presents access to the callback registered with time
	callbackRef        *time.Timer
	callbackExpectedAt time.Time
	rwL                *sync.RWMutex
}

func NewTimedCallback() TimedCallback {
	return TimedCallback{
		callbackRef:        nil,
		rwL:                &sync.RWMutex{},
		callbackExpectedAt: time.Now(),
	}
}

func (t TimedCallback) IsExpectingCallback() bool {
	return t.callbackRef != nil
}
func (t TimedCallback) GetCallbackExpectedAt() time.Time {
	return t.callbackExpectedAt
}
func (t *TimedCallback) Stop() {
	t.rwL.Lock()
	if t.callbackRef != nil && !t.callbackRef.Stop() {
		<-t.callbackRef.C
	}
	t.callbackRef = nil
	t.rwL.Unlock()
}
func (t *TimedCallback) CallMeBackAfter(d time.Duration, f func()) {
	t.CallMeBackAt(time.Now().Add(d), f)
}
func (t *TimedCallback) CallMeBackAt(at time.Time, f func()) bool {
	now := time.Now()
	if at.After(now) {
		t.Stop()
		t.rwL.Lock()
		t.callbackRef = time.AfterFunc(at.Sub(now), func() {
			t.rwL.Lock()
			t.callbackRef = nil
			t.rwL.Unlock()
			f()
		})
		t.callbackExpectedAt = at

		t.rwL.Unlock()
		return true
	}
	return false
}
func (t *TimedCallback) CallMeBackIfEarlierThanCurrent(at time.Time, f func()) bool {
	t.rwL.RLock()
	if !t.IsExpectingCallback() || at.Before(t.GetCallbackExpectedAt()) {
		t.rwL.RUnlock()
		t.CallMeBackAt(at, f)
		return true
	}
	t.rwL.RUnlock()
	return false
}
