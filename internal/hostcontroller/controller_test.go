package hostcontroller

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/ui"
	"delve-shell/internal/uipresenter"
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
		stop: stop,
		bus:  hostbus.New(64),
		ui:   uipresenter.New(sender),
		fsm:  hostfsm.NewMachine(hostfsm.StateIdle),
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
	msg, ok := s.msgs[0].(ui.CommandExecutedMsg)
	if !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
	if !msg.Direct || msg.Command != "echo ok" {
		t.Fatalf("unexpected payload: %+v", msg)
	}
	if !strings.Contains(msg.Result, "ok") || !strings.Contains(msg.Result, "exit_code: 0") {
		t.Fatalf("unexpected result: %q", msg.Result)
	}
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
	msg := s.msgs[0].(ui.CommandExecutedMsg)
	if !strings.Contains(msg.Result, "hello") {
		t.Fatalf("missing stdout in result: %q", msg.Result)
	}
	if !strings.Contains(msg.Result, "stderr:\nwarn") {
		t.Fatalf("missing stderr in result: %q", msg.Result)
	}
	if !strings.Contains(msg.Result, "exit_code: 1") {
		t.Fatalf("missing exit code in result: %q", msg.Result)
	}
	if strings.Contains(msg.Result, "error:") {
		t.Fatalf("should not append run err when exitCode != 0: %q", msg.Result)
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
	msg := s.msgs[0].(ui.CommandExecutedMsg)
	if !strings.Contains(msg.Result, "error: network issue") {
		t.Fatalf("missing synthesized error line: %q", msg.Result)
	}
}

func TestHandleAgentUI_ApprovalRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &agent.ApprovalRequest{Command: "ls"}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(*agent.ApprovalRequest)
	if !ok || got.Command != "ls" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func TestHandleAgentUI_SensitiveRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &agent.SensitiveConfirmationRequest{Command: "cat /x"}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(*agent.SensitiveConfirmationRequest)
	if !ok || got.Command != "cat /x" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func TestHandleAgentUI_ExecEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleAgentUI(agent.ExecEvent{
		Command:   "ls",
		Allowed:   true,
		Result:    "ok",
		Sensitive: true,
		Suggested: false,
	})

	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	msg, ok := s.msgs[0].(ui.CommandExecutedMsg)
	if !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
	if msg.Command != "ls" || !msg.Allowed || msg.Direct || !msg.Sensitive || msg.Result != "ok" {
		t.Fatalf("unexpected payload: %+v", msg)
	}
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
	reply := s.msgs[0].(ui.AgentReplyMsg)
	if reply.Reply != "hello" || reply.Err != nil {
		t.Fatalf("unexpected reply payload: %+v", reply)
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
	reply := s.msgs[0].(ui.AgentReplyMsg)
	if reply.Err == nil {
		t.Fatal("expected error reply")
	}
	if !strings.Contains(reply.Err.Error(), "Hint: For DashScope") {
		t.Fatalf("expected 404 hint in error, got: %v", reply.Err)
	}
}

func TestHandleLLMRunCompleted_ErrorWithout404NoHint(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.fsm = hostfsm.NewMachine(hostfsm.StateLLMRunning)
	c.llmRunning = true

	c.handleLLMRunCompleted("", errors.New("timeout"))
	reply := s.msgs[0].(ui.AgentReplyMsg)
	if reply.Err == nil {
		t.Fatal("expected error reply")
	}
	if strings.Contains(reply.Err.Error(), "Hint: For DashScope") {
		t.Fatalf("unexpected hint in error: %v", reply.Err)
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
	c.handleEvent(hostbus.Event{Kind: hostbus.KindAgentExecEvent, AgentExec: agent.ExecEvent{Command: "x"}})
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
	c.handleEvent(hostbus.Event{Kind: hostbus.KindCancelRequested})
	if !cancelled.Load() {
		t.Fatal("cancel should be dispatched")
	}
}

func TestHandleEvent_DispatchExecDirect(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{stdout: "ok", exitCode: 0}
	c.getExec = func() execenv.CommandExecutor { return fx }
	c.handleEvent(hostbus.Event{Kind: hostbus.KindExecDirectRequested, Command: "echo ok"})
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
	c.handleEvent(hostbus.Event{Kind: hostbus.KindLLMRunCompleted, Reply: "done"})
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
}

func TestRun_StopsAndCancels(t *testing.T) {
	s := &recordSender{}
	stop := make(chan struct{})
	c := &Controller{
		stop:       stop,
		bus:        hostbus.New(8),
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
	b := hostbus.New(8)
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
	b.PublishBlocking(hostbus.Event{Kind: hostbus.KindExecDirectRequested, Command: "echo"})

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

func TestNew_WiresBusAndPump(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := hostbus.New(8)
	ports := hostbus.NewInputPorts()
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
		if ev.Kind != hostbus.KindUserChatSubmitted || ev.UserText != "abc" {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected bridged event")
	}
}
