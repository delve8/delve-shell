package inputlifecycle

import (
	"errors"
	"testing"

	"delve-shell/internal/inputlifecycletype"
)

type stubProcessor struct {
	match bool
	res   inputlifecycletype.ProcessResult
	err   error
	calls int
}

func (s *stubProcessor) CanProcess(inputlifecycletype.InputSubmission) bool { return s.match }

func (s *stubProcessor) Process(inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	s.calls++
	return s.res, s.err
}

func TestRouterRouteFirstMatchWins(t *testing.T) {
	first := &stubProcessor{match: false}
	second := &stubProcessor{match: true, res: inputlifecycletype.ConsumedResult()}
	third := &stubProcessor{match: true, res: inputlifecycletype.ConsumedResult()}

	router := NewRouter(first, second, third)
	_, err := router.Route(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionChat})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if first.calls != 0 {
		t.Fatalf("first processor should not process, got %d calls", first.calls)
	}
	if second.calls != 1 {
		t.Fatalf("second processor calls = %d want 1", second.calls)
	}
	if third.calls != 0 {
		t.Fatalf("third processor should not be reached, got %d calls", third.calls)
	}
}

func TestRouterRouteNoMatch(t *testing.T) {
	router := NewRouter(&stubProcessor{match: false})
	_, err := router.Route(inputlifecycletype.InputSubmission{Kind: inputlifecycletype.SubmissionControl})
	if !errors.Is(err, ErrNoProcessorMatched) {
		t.Fatalf("Route() error = %v want %v", err, ErrNoProcessorMatched)
	}
}
