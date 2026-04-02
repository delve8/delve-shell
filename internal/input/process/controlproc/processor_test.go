package controlproc

import (
	"errors"
	"testing"

	"delve-shell/internal/input/lifecycletype"
)

type stubContexts struct {
	ctx inputlifecycletype.ControlContext
}

func (s stubContexts) ControlContext() inputlifecycletype.ControlContext { return s.ctx }

type stubExecutor struct {
	action inputlifecycletype.ControlAction
	calls  int
	res    inputlifecycletype.ProcessResult
	err    error
}

func (s *stubExecutor) ExecuteControl(action inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error) {
	s.calls++
	s.action = action
	return s.res, s.err
}

func TestProcessorCanProcess(t *testing.T) {
	p := New(nil, nil)
	if !p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionControl}) {
		t.Fatal("control submission should match control processor")
	}
	if p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionChat}) {
		t.Fatal("chat submission should not match control processor")
	}
}

func TestProcessorProcessEsc_CommandExecuting(t *testing.T) {
	exec := &stubExecutor{res: inputlifecycletype.ConsumedResult()}
	p := New(stubContexts{ctx: inputlifecycletype.ControlContext{CommandExecuting: true, WaitingForAI: true}}, exec)
	_, err := p.Process(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionControl,
		ControlSignal: inputlifecycletype.ControlSignalEsc,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if exec.action != inputlifecycletype.ControlCancelCommandExecution {
		t.Fatalf("ExecuteControl action = %q want cancel_command_execution", exec.action)
	}
}

func TestProcessorProcessEsc(t *testing.T) {
	exec := &stubExecutor{res: inputlifecycletype.ConsumedResult()}
	p := New(stubContexts{ctx: inputlifecycletype.ControlContext{WaitingForAI: true}}, exec)

	_, err := p.Process(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionControl,
		ControlSignal: inputlifecycletype.ControlSignalEsc,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("ExecuteControl calls = %d want 1", exec.calls)
	}
	if exec.action != inputlifecycletype.ControlCancelProcessing {
		t.Fatalf("ExecuteControl action = %q want cancel_processing", exec.action)
	}
}

func TestProcessorProcessQuit(t *testing.T) {
	exec := &stubExecutor{res: inputlifecycletype.ConsumedResult()}
	p := New(nil, exec)

	_, err := p.Process(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionControl,
		ControlSignal: inputlifecycletype.ControlSignalQuit,
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if exec.action != inputlifecycletype.ControlQuit {
		t.Fatalf("ExecuteControl action = %q want quit", exec.action)
	}
}

func TestProcessorProcessUnknownSignal(t *testing.T) {
	p := New(nil, nil)
	_, err := p.Process(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionControl})
	if !errors.Is(err, ErrUnknownControlSignal) {
		t.Fatalf("Process() error = %v want %v", err, ErrUnknownControlSignal)
	}
}
