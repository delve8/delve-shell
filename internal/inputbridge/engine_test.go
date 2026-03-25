package inputbridge

import (
	"testing"

	"delve-shell/internal/inputlifecycletype"
)

func TestNewEngineRoutesChat(t *testing.T) {
	sink := &stubSink{accepts: true}
	engine := NewEngine(sink, controlContextProvider{}, nil)

	res, handled, err := engine.SubmitEnter("hello", 0)
	if err != nil {
		t.Fatalf("SubmitEnter() error = %v", err)
	}
	if !handled {
		t.Fatal("SubmitEnter() should be handled")
	}
	if sink.action.Kind != "submit" {
		t.Fatalf("unexpected action kind: %q", sink.action.Kind)
	}
	if !res.WaitingForAI {
		t.Fatal("expected chat result to mark WaitingForAI")
	}
}

func TestNewEngineRoutesEscControl(t *testing.T) {
	sink := &stubSink{accepts: true}
	engine := NewEngine(sink, controlContextProvider{
		ctx: inputlifecycletype.ControlContext{WaitingForAI: true},
	}, nil)

	_, err := engine.SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
	if err != nil {
		t.Fatalf("SubmitControl() error = %v", err)
	}
	if sink.action.Kind != "cancel_requested" {
		t.Fatalf("unexpected action kind: %q", sink.action.Kind)
	}
}
