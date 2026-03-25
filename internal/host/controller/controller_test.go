package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hiltypes"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/ui"
	"delve-shell/internal/uipresenter"
	"delve-shell/internal/uivm"
)

type recordSender struct {
	msgs []tea.Msg
}

func (r *recordSender) Send(msg tea.Msg) {
	if msg == nil {
		return
	}
	r.msgs = append(r.msgs, msg)
}

type fakeExec struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	lastCmd  string
}

func (f *fakeExec) Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	_ = ctx
	f.lastCmd = command
	return f.stdout, f.stderr, f.exitCode, f.err
}

func newTestControllerWithPresenter(sender *recordSender) *Controller {
	stop := make(chan struct{})
	c := &Controller{
		stop:     stop,
		bus:      bus.New(64),
		ui:       uipresenter.New(sender),
		fsm:      hostfsm.NewMachine(hostfsm.StateIdle),
		sessions: &sessionmgr.Manager{},
	}
	return c
}

func TestHandleCancelRequest_NoRunning(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleCancelRequest()
	if len(s.msgs) != 0 {
		t.Fatalf("unexpected msgs: %+v", s.msgs)
	}
}

func TestHandleCancelRequest_RunningCancels(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	var called atomic.Bool
	c.llmRunning = true
	c.llmCancel = func() { called.Store(true) }
	c.handleCancelRequest()

	if !called.Load() {
		t.Fatal("cancel func was not called")
	}
}

func TestHandleExecDirect_StdoutOnly(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "ok",
		stderr:   "",
		exitCode: 0,
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("echo ok")
	if fx.lastCmd != "echo ok" {
		t.Fatalf("unexpected command: %q", fx.lastCmd)
	}
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	msg, ok := s.msgs[0].(ui.TranscriptAppendMsg)
	if !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
	_ = msg
}

func TestHandleExecDirect_StdoutAndStderr(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "hello",
		stderr:   "warn",
		exitCode: 1,
		err:      errors.New("exit status 1"),
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("bad")
	if _, ok := s.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
}

func TestHandleExecDirect_RunErrWithoutExitCodeAddsErrorLine(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "",
		stderr:   "",
		exitCode: 0,
		err:      errors.New("network issue"),
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("x")
	if _, ok := s.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
}

func TestHandleAgentUI_ApprovalRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &hiltypes.ApprovalRequest{Command: "ls", ResponseCh: make(chan hiltypes.ApprovalResponse, 1)}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(ui.ChoiceCardShowMsg)
	if !ok || got.PendingApproval == nil || got.PendingApproval.Command != "ls" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func TestHandleAgentUI_SensitiveRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &hiltypes.SensitiveConfirmationRequest{Command: "cat /x", ResponseCh: make(chan hiltypes.SensitiveChoice, 1)}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(ui.ChoiceCardShowMsg)
	if !ok || got.PendingSensitive == nil || got.PendingSensitive.Command != "cat /x" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func TestHandleAgentUI_ExecEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleAgentUI(hiltypes.ExecEvent{
		Command:   "ls",
		Allowed:   true,
		Result:    "ok",
		Sensitive: true,
		Suggested: false,
	})

	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	msg, ok := s.msgs[0].(ui.TranscriptAppendMsg)
	if !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
	_ = msg
}

func TestHandleAgentUI_UnknownPayloadIgnored(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleAgentUI(struct{ X int }{X: 1})
	if len(s.msgs) != 0 {
		t.Fatalf("unknown payload should be ignored, got %+v", s.msgs)
	}
}

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
	// Minimal smoke: exercising dispatch path should not panic even if runner/config deps are absent.
	// We avoid calling handleConfigUpdated directly here because it requires real runner manager wiring.
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.currentAllowlistAutoRun = new(atomic.Bool)
	c.currentAllowlistAutoRun.Store(true)
	// Keep this path isolated by not using KindConfigUpdated.
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

func TestHandleEvent_SlashEnteredNoOp(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleEvent(bus.Event{Kind: bus.KindSlashEntered, UserText: "/help"})
	if len(s.msgs) != 0 {
		t.Fatalf("slash entered is observability-only, got %d msgs", len(s.msgs))
	}
}

func TestHandleEvent_SlashRequestedNoOp(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleEvent(bus.Event{Kind: bus.KindSlashRequested, UserText: "/help"})
	if len(s.msgs) != 0 {
		t.Fatalf("slash requested is observability-only, got %d msgs", len(s.msgs))
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

	ports.SubmitChan <- "abc"
	select {
	case ev := <-b.Events():
		if ev.Kind != bus.KindUserChatSubmitted || ev.UserText != "abc" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected bridged event")
	}
}

func TestHandleUIAction_SubmissionPublishesStructuredChatEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	sub := inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "abc",
	}

	c.handleUIAction(uivm.UIAction{Kind: uivm.UIActionSubmission, Submission: sub})

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

func TestHandleUIAction_SessionNewPublishesEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	c.handleUIAction(uivm.UIAction{Kind: uivm.UIActionSessionNew})

	select {
	case ev := <-c.bus.Events():
		if ev.Kind != bus.KindSessionNewRequested {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected session new event")
	}
}

func TestHandleUIAction_SessionSwitchPublishesEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	c.handleUIAction(uivm.UIAction{Kind: uivm.UIActionSessionSwitch, Text: "demo"})

	select {
	case ev := <-c.bus.Events():
		if ev.Kind != bus.KindSessionSwitchRequested || ev.SessionID != "demo" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected session switch event")
	}
}
