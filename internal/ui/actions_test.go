package ui

import (
	"testing"
	"time"

	"delve-shell/internal/host/cmd"
)

func TestCommandChannelSender_CancelBlocksWhenBufferFull(t *testing.T) {
	ch := make(chan hostcmd.Command, 1)
	ch <- hostcmd.SessionNew{}
	s := NewCommandChannelSender(ch)
	sent := make(chan struct{})
	go func() {
		if !s.Send(hostcmd.CancelRequested{}) {
			t.Error("cancel Send should return true")
		}
		close(sent)
	}()
	select {
	case <-sent:
		t.Fatal("cancel Send should block until the buffer is drained")
	case <-time.After(50 * time.Millisecond):
	}
	first := <-ch
	if _, ok := first.(hostcmd.SessionNew); !ok {
		t.Fatalf("expected SessionNew, got %T", first)
	}
	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cancel Send after drain")
	}
	second := <-ch
	if _, ok := second.(hostcmd.CancelRequested); !ok {
		t.Fatalf("expected CancelRequested, got %T", second)
	}
}
