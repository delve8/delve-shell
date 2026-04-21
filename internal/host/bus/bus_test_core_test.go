package bus

import (
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func TestNew_DefaultCapacity(t *testing.T) {
	b := New(0)
	if b == nil {
		t.Fatal("bus is nil")
	}
	if cap(b.events) != 128 {
		t.Fatalf("want default event cap 128, got %d", cap(b.events))
	}
	if cap(b.uiMsgs) != 256 {
		t.Fatalf("want ui cap 256, got %d", cap(b.uiMsgs))
	}
}

func TestPublishAndEvents(t *testing.T) {
	b := New(4)
	ok := b.Publish(Event{Kind: KindUserChatSubmitted, UserText: "hello"})
	if !ok {
		t.Fatal("publish failed unexpectedly")
	}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindUserChatSubmitted || ev.UserText != "hello" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestPublishQueueFull(t *testing.T) {
	b := New(1)
	if !b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("first publish should succeed")
	}
	if b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("second publish should fail when queue full")
	}
}

func TestPublishBlockingUnblocksAfterReceive(t *testing.T) {
	b := New(1)
	if !b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("seed publish failed")
	}
	done := make(chan struct{})
	go func() {
		b.PublishBlocking(Event{Kind: KindCancelRequested})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("publish blocking should not complete before dequeue")
	case <-time.After(80 * time.Millisecond):
	}

	_ = mustRecvEvent(t, b.Events())
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocking did not unblock")
	}
}

func TestEnqueueUI_NilMessage(t *testing.T) {
	b := New(2)
	if b.EnqueueUI(nil) {
		t.Fatal("expected false for nil UI message")
	}
}

func TestEnqueueUI_QueueFull(t *testing.T) {
	b := New(2)
	for i := 0; i < cap(b.uiMsgs); i++ {
		if !b.EnqueueUI(ui.TranscriptAppendMsg{}) {
			t.Fatalf("enqueue failed unexpectedly at index %d", i)
		}
	}
	if b.EnqueueUI(ui.TranscriptAppendMsg{}) {
		t.Fatal("enqueue should fail when ui queue full")
	}
}

func TestEnqueueUIBlocking_IgnoresNil(t *testing.T) {
	b := New(2)
	b.EnqueueUIBlocking(nil)
	select {
	case <-b.uiMsgs:
		t.Fatal("nil should not enqueue")
	default:
	}
}

func TestInputPortsDefaults(t *testing.T) {
	in := NewInputPorts()
	if cap(in.SubmissionChan) != 8 ||
		cap(in.ConfigUpdatedChan) != 8 ||
		cap(in.CancelRequestChan) != 8 ||
		cap(in.RemoteOnChan) != 4 ||
		cap(in.RemoteOffChan) != 4 ||
		cap(in.RemoteAuthRespChan) != 4 ||
		cap(in.AgentUIChan) != 64 {
		t.Fatalf("unexpected capacities: %+v", in)
	}
}

func TestStartUIPump_NoProgram(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	var currentP atomic.Pointer[tea.Program]
	StartUIPump(stop, b, &currentP)

	if !b.EnqueueUI(ui.TranscriptAppendMsg{}) {
		t.Fatal("enqueue failed unexpectedly")
	}
	time.Sleep(20 * time.Millisecond)
}

func TestStartUIPump_Stop(t *testing.T) {
	stop := make(chan struct{})

	b := New(8)
	var currentP atomic.Pointer[tea.Program]
	StartUIPump(stop, b, &currentP)
	close(stop)
	time.Sleep(20 * time.Millisecond)

	_ = b.EnqueueUI(ui.TranscriptAppendMsg{})
}

func TestPublishHook_SuccessAndDrop(t *testing.T) {
	var nAccepted, nDropped atomic.Int32
	h := func(e Event, accepted bool) {
		_ = e
		if accepted {
			nAccepted.Add(1)
		} else {
			nDropped.Add(1)
		}
	}
	b := New(1, WithPublishHook(h))
	if !b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("first publish")
	}
	if nAccepted.Load() != 1 || nDropped.Load() != 0 {
		t.Fatalf("after first: accepted=%d dropped=%d", nAccepted.Load(), nDropped.Load())
	}
	if b.Publish(Event{Kind: KindCancelRequested}) {
		t.Fatal("second publish should fail when queue full")
	}
	if nAccepted.Load() != 1 || nDropped.Load() != 1 {
		t.Fatalf("after drop: accepted=%d dropped=%d", nAccepted.Load(), nDropped.Load())
	}
}
