package controller

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"delve-shell/internal/agent"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/ui"
	"delve-shell/internal/uipresenter"
)

func TestHandleLLMRunCompleted_Success(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.fsm = hostfsm.NewMachine(hostfsm.StateLLMRunning)
	c.llmRunning = true
	var cancelled atomic.Bool
	c.llmCancel = func() { cancelled.Store(true) }

	c.handleLLMRunCompleted("hello", nil)

	if !cancelled.Load() {
		t.Fatal("expected cancel cleanup to run")
	}
	if c.llmRunning {
		t.Fatal("llmRunning should be false")
	}
	if c.llmCancel != nil {
		t.Fatal("llmCancel should be nil")
	}
	if c.fsm.State() != hostfsm.StateIdle {
		t.Fatalf("fsm should return to idle, got %q", c.fsm.State())
	}
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	if _, ok := s.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
}

func TestHandleLLMRunCompleted_ErrorWith404Hint(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.fsm = hostfsm.NewMachine(hostfsm.StateLLMRunning)
	c.llmRunning = true

	c.handleLLMRunCompleted("", errors.New("request failed: 404 not found"))
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	if _, ok := s.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
}

func TestHandleLLMRunCompleted_ErrorWithout404NoHint(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.fsm = hostfsm.NewMachine(hostfsm.StateLLMRunning)
	c.llmRunning = true

	c.handleLLMRunCompleted("", errors.New("timeout"))
	if _, ok := s.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
}

func TestHandleEvent_DispatchConfigUpdated(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.currentAllowlistAutoRun = new(atomic.Bool)
	c.currentAllowlistAutoRun.Store(true)
	c.handleEvent(bus.Event{Kind: bus.KindAgentExecEvent, AgentExec: agent.ExecEvent{Command: "x"}})
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
}

func TestHandleEvent_DispatchCancel(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	var cancelled atomic.Bool
	c.llmRunning = true
	c.llmCancel = func() { cancelled.Store(true) }
	c.handleEvent(bus.Event{Kind: bus.KindCancelRequested})
	if !cancelled.Load() {
		t.Fatal("cancel should be dispatched")
	}
}

func TestHandleEvent_DispatchExecDirect(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{stdout: "ok", exitCode: 0}
	c.getExec = func() execenv.CommandExecutor { return fx }
	c.handleEvent(bus.Event{Kind: bus.KindExecDirectRequested, Command: "echo ok"})
	if fx.lastCmd != "echo ok" {
		t.Fatalf("unexpected cmd: %q", fx.lastCmd)
	}
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
}

func TestHandleEvent_DispatchLLMCompleted(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.fsm = hostfsm.NewMachine(hostfsm.StateLLMRunning)
	c.llmRunning = true
	c.handleEvent(bus.Event{Kind: bus.KindLLMRunCompleted, Reply: "done"})
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
}

func TestRun_StopsAndCancels(t *testing.T) {
	s := &recordSender{}
	stop := make(chan struct{})
	c := &Controller{
		stop:       stop,
		bus:        bus.New(8),
		ui:         uipresenter.New(s),
		fsm:        hostfsm.NewMachine(hostfsm.StateIdle),
		llmRunning: true,
	}
	var cancelled atomic.Bool
	c.llmCancel = func() { cancelled.Store(true) }

	done := make(chan struct{})
	go func() {
		c.run()
		close(done)
	}()

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run loop did not stop")
	}
	if !cancelled.Load() {
		t.Fatal("expected cancel on stop")
	}
}

func TestRun_ProcessesBusEvents(t *testing.T) {
	s := &recordSender{}
	stop := make(chan struct{})
	defer close(stop)
	b := bus.New(8)
	fx := &fakeExec{stdout: "ok", exitCode: 0}
	c := &Controller{
		stop:    stop,
		bus:     b,
		ui:      uipresenter.New(s),
		getExec: func() execenv.CommandExecutor { return fx },
		fsm:     hostfsm.NewMachine(hostfsm.StateIdle),
	}

	done := make(chan struct{})
	go func() {
		c.run()
		close(done)
	}()
	b.PublishBlocking(bus.Event{Kind: bus.KindExecDirectRequested, Command: "echo"})

	deadline := time.After(2 * time.Second)
	for len(s.msgs) == 0 {
		select {
		case <-deadline:
			t.Fatal("no message emitted")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestSyncCurrentSessionPath_NoHook(t *testing.T) {
	c := &Controller{}
	c.SyncCurrentSessionPath()
}

func TestHandleEvent_UnknownKindNoOp(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleEvent(bus.Event{Kind: bus.Kind("nosuch_kind")})
	if len(s.msgs) != 0 {
		t.Fatalf("unknown kind should not dispatch, got %d msgs", len(s.msgs))
	}
}

func TestHandleEvent_OnEventDispatchCalled(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	var saw atomic.Bool
	c.onEventDispatch = func(e bus.Event) {
		if e.Kind == bus.KindCancelRequested {
			saw.Store(true)
		}
	}
	c.llmRunning = true
	c.llmCancel = func() {}
	c.handleEvent(bus.Event{Kind: bus.KindCancelRequested})
	if !saw.Load() {
		t.Fatal("onEventDispatch not invoked")
	}
}
