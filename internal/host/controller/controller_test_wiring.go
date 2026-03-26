package controller

import (
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/bus"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/inputlifecycletype"
)

func TestNew_WiresBusAndPump(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := bus.New(8)
	ports := bus.NewInputPorts()
	var p atomic.Pointer[tea.Program]
	var auto atomic.Bool

	c := New(Options{
		Stop:                    stop,
		Bus:                     b,
		Inputs:                  ports,
		CurrentP:                &p,
		CurrentAllowlistAutoRun: &auto,
	})
	if c == nil {
		t.Fatal("controller is nil")
	}
	if c.fsm == nil {
		t.Fatal("fsm not initialized")
	}

	ports.SubmissionChan <- inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "abc",
	}
	select {
	case ev := <-b.Events():
		if ev.Kind != bus.KindUserChatSubmitted || ev.UserText != "abc" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected bridged event")
	}
}

func TestHandleCommand_SubmissionPublishesStructuredChatEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	sub := inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "abc",
	}

	c.handleCommand(hostcmd.Submission{Submission: sub})

	select {
	case ev := <-c.bus.Events():
		if ev.Kind != bus.KindUserChatSubmitted || ev.UserText != "abc" {
			t.Fatalf("unexpected event header: %+v", ev)
		}
		if ev.Submission != sub {
			t.Fatalf("submission mismatch: got %#v want %#v", ev.Submission, sub)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected chat event")
	}
}

func TestHandleCommand_SessionNewPublishesEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	c.handleCommand(hostcmd.SessionNew{})

	select {
	case ev := <-c.bus.Events():
		if ev.Kind != bus.KindSessionNewRequested {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected session new event")
	}
}

func TestHandleCommand_SessionSwitchPublishesEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	c.handleCommand(hostcmd.SessionSwitch{SessionID: "demo"})

	select {
	case ev := <-c.bus.Events():
		if ev.Kind != bus.KindSessionSwitchRequested || ev.SessionID != "demo" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected session switch event")
	}
}
