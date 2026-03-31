package controller

import (
	"errors"
	"sync/atomic"
	"testing"

	"delve-shell/internal/hiltypes"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/ui"
)

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
