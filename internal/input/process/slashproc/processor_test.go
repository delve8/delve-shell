package slashproc

import (
	"errors"
	"testing"

	"delve-shell/internal/input/lifecycletype"
)

type stubExecutor struct {
	req   ExecutionRequest
	calls int
	res   inputlifecycletype.ProcessResult
	err   error
}

func (s *stubExecutor) ExecuteSlash(req ExecutionRequest) (inputlifecycletype.ProcessResult, error) {
	s.calls++
	s.req = req
	return s.res, s.err
}

func TestProcessorCanProcess(t *testing.T) {
	p := New(nil)
	if !p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionSlash}) {
		t.Fatal("slash submission should match slash processor")
	}
	if p.CanProcess(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionChat}) {
		t.Fatal("chat submission should not match slash processor")
	}
}

func TestProcessorProcess(t *testing.T) {
	exec := &stubExecutor{res: inputlifecycletype.ConsumedResult()}
	p := New(exec)

	_, err := p.Process(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionSlash,
		RawText:       "/access New",
		InputLine:     " /access New ",
		SelectedIndex: 2,
		SelectedCmd:   "/access 10.0.0.1",
		SelectedFill:  "/access 10.0.0.1",
		SelectedExec:  "/access jump",
	})
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("ExecuteSlash calls = %d want 1", exec.calls)
	}
	if exec.req.RawText != "/access New" {
		t.Fatalf("RawText = %q want /access New", exec.req.RawText)
	}
	if exec.req.InputLine != " /access New " {
		t.Fatalf("InputLine = %q want raw input line", exec.req.InputLine)
	}
	if exec.req.SelectedIndex != 2 {
		t.Fatalf("SelectedIndex = %d want 2", exec.req.SelectedIndex)
	}
	if exec.req.SelectedCmd != "/access 10.0.0.1" {
		t.Fatalf("SelectedCmd = %q want /access 10.0.0.1", exec.req.SelectedCmd)
	}
	if exec.req.SelectedFill != "/access 10.0.0.1" {
		t.Fatalf("SelectedFill = %q want /access 10.0.0.1", exec.req.SelectedFill)
	}
	if exec.req.SelectedExec != "/access jump" {
		t.Fatalf("SelectedExec = %q want /access jump", exec.req.SelectedExec)
	}
}

func TestProcessorProcessWithoutExecutor(t *testing.T) {
	p := New(nil)
	_, err := p.Process(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionSlash})
	if !errors.Is(err, ErrSlashExecutorMissing) {
		t.Fatalf("Process() error = %v want %v", err, ErrSlashExecutorMissing)
	}
}
