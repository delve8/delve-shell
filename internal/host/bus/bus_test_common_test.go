package bus

import (
	"testing"
	"time"
)

func mustRecvEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bus event")
		return Event{}
	}
}
