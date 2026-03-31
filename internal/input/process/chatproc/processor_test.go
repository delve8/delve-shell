package chatproc

import (
	"errors"
	"testing"

	"delve-shell/internal/input/lifecycletype"
)

type stubExecutor struct {
	sub   inputlifecycletype.InputSubmission
	calls int
	res   inputlifecycletype.ProcessResult
	err   error
}

func (s *stubExecutor) ExecuteChat(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	s.calls++
	s.sub = sub
	return s.res, s.err
}

func TestProcessorCanProcess(t *testing.T) {
	p := New(nil)
	if !p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionChat}) {
		t.Fatal("chat submission should match chat processor")
	}
	if p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionSlash}) {
		t.Fatal("slash submission should not match chat processor")
	}
}

func TestProcessorProcess(t *testing.T) {
	exec := &stubExecutor{res: inputlifecycletype.ConsumedResult()}
	p := New(exec)

	_, err := p.Process(inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		RawText: "hello",
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("ExecuteChat calls = %d want 1", exec.calls)
	}
	if exec.sub.RawText != "hello" {
		t.Fatalf("RawText = %q want hello", exec.sub.RawText)
	}
}

func TestProcessorProcessWithoutExecutor(t *testing.T) {
	p := New(nil)
	_, err := p.Process(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionChat})
	if !errors.Is(err, ErrChatExecutorMissing) {
		t.Fatalf("Process() error = %v want %v", err, ErrChatExecutorMissing)
	}
}
