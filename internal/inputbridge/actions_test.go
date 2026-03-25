package inputbridge

import (
	"errors"
	"testing"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/uivm"
)

type stubSink struct {
	action  uivm.UIAction
	calls   int
	accepts bool
}

func (s *stubSink) Send(action uivm.UIAction) bool {
	s.calls++
	s.action = action
	return s.accepts
}

func TestChatActionExecutor(t *testing.T) {
	sink := &stubSink{accepts: true}
	exec := ChatActionExecutor{Sink: sink}

	got, err := exec.ExecuteChat(inputlifecycletype.InputSubmission{RawText: "hello"})
	if err != nil {
		t.Fatalf("ExecuteChat() error = %v", err)
	}
	if sink.action.Kind != uivm.UIActionSubmit || sink.action.Text != "hello" {
		t.Fatalf("unexpected action: %#v", sink.action)
	}
	if !got.WaitingForAI {
		t.Fatal("expected chat bridge to mark WaitingForAI")
	}
}

func TestControlActionExecutorCancel(t *testing.T) {
	sink := &stubSink{accepts: true}
	exec := ControlActionExecutor{Sink: sink}

	_, err := exec.ExecuteControl(inputlifecycletype.ControlCancelProcessing)
	if err != nil {
		t.Fatalf("ExecuteControl() error = %v", err)
	}
	if sink.action.Kind != uivm.UIActionCancelRequested {
		t.Fatalf("unexpected action kind: %q", sink.action.Kind)
	}
}

func TestBridgeRejected(t *testing.T) {
	sink := &stubSink{accepts: false}
	exec := ChatActionExecutor{Sink: sink}
	_, err := exec.ExecuteChat(inputlifecycletype.InputSubmission{RawText: "hello"})
	if !errors.Is(err, ErrActionRejected) {
		t.Fatalf("ExecuteChat() error = %v want %v", err, ErrActionRejected)
	}
}
