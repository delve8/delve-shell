package inputlifecycle

import (
	"testing"

	"delve-shell/internal/inputlifecycletype"
)

type stubPreInput struct {
	sub inputlifecycletype.InputSubmission
	ok  bool
}

func (s stubPreInput) OnInputChanged(current string) inputlifecycletype.PreInputState {
	return inputlifecycletype.PreInputState{InputValue: current}
}

func (s stubPreInput) OnEnter(current string, selectedIndex int) (inputlifecycletype.InputSubmission, bool) {
	return s.sub, s.ok
}

type stubRouter struct {
	sub   inputlifecycletype.InputSubmission
	calls int
	res   inputlifecycletype.ProcessResult
	err   error
}

func (s *stubRouter) Route(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	s.calls++
	s.sub = sub
	return s.res, s.err
}

func TestEngineSubmitEnter(t *testing.T) {
	router := &stubRouter{res: inputlifecycletype.ConsumedResult()}
	engine := NewEngine(stubPreInput{
		sub: inputlifecycletype.InputSubmission{
			Kind:    inputlifecycletype.SubmissionChat,
			RawText: "hello",
		},
		ok: true,
	}, router)

	_, handled, err := engine.SubmitEnter("hello", 0)
	if err != nil {
		t.Fatalf("SubmitEnter() error = %v", err)
	}
	if !handled {
		t.Fatal("SubmitEnter() should report handled=true")
	}
	if router.calls != 1 {
		t.Fatalf("router calls = %d want 1", router.calls)
	}
	if router.sub.RawText != "hello" {
		t.Fatalf("router submission RawText = %q want hello", router.sub.RawText)
	}
}

func TestEngineSubmitEnterNoSubmission(t *testing.T) {
	router := &stubRouter{}
	engine := NewEngine(stubPreInput{ok: false}, router)

	_, handled, err := engine.SubmitEnter("   ", 0)
	if err != nil {
		t.Fatalf("SubmitEnter() error = %v", err)
	}
	if handled {
		t.Fatal("SubmitEnter() should report handled=false")
	}
	if router.calls != 0 {
		t.Fatalf("router calls = %d want 0", router.calls)
	}
}

func TestEngineSubmitControl(t *testing.T) {
	router := &stubRouter{res: inputlifecycletype.ConsumedResult()}
	engine := NewEngine(nil, router)

	_, err := engine.SubmitControl(inputlifecycletype.ControlSignalEsc, inputlifecycletype.SourceKeyboardSignal)
	if err != nil {
		t.Fatalf("SubmitControl() error = %v", err)
	}
	if router.calls != 1 {
		t.Fatalf("router calls = %d want 1", router.calls)
	}
	if router.sub.Kind != inputlifecycletype.SubmissionControl {
		t.Fatalf("router submission kind = %q want control", router.sub.Kind)
	}
	if router.sub.ControlSignal != inputlifecycletype.ControlSignalEsc {
		t.Fatalf("router control signal = %q want esc", router.sub.ControlSignal)
	}
	if router.sub.Source != inputlifecycletype.SourceKeyboardSignal {
		t.Fatalf("router source = %q want keyboard_signal", router.sub.Source)
	}
}

func TestEngineRouteSubmission(t *testing.T) {
	router := &stubRouter{res: inputlifecycletype.ConsumedResult()}
	engine := NewEngine(nil, router)

	sub := inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionSlash,
		Source:        inputlifecycletype.SourceSlashEarlyEnter,
		RawText:       "/help",
		InputLine:     " /help ",
		SelectedIndex: 1,
	}
	_, handled, err := engine.RouteSubmission(sub)
	if err != nil {
		t.Fatalf("RouteSubmission() error = %v", err)
	}
	if !handled {
		t.Fatal("RouteSubmission() should report handled=true")
	}
	if router.sub != sub {
		t.Fatalf("router submission = %#v want %#v", router.sub, sub)
	}
}
