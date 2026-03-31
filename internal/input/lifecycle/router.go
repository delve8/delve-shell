package inputlifecycle

import (
	"errors"

	"delve-shell/internal/input/lifecycletype"
)

// ErrNoProcessorMatched is returned when no processor claims a submission.
var ErrNoProcessorMatched = errors.New("input lifecycle: no processor matched submission")

// Router dispatches submissions to the first matching processor.
type Router struct {
	processors []inputlifecycletype.SubmissionProcessor
}

// NewRouter creates a router with processors evaluated in the provided order.
func NewRouter(processors ...inputlifecycletype.SubmissionProcessor) Router {
	cloned := append([]inputlifecycletype.SubmissionProcessor(nil), processors...)
	return Router{processors: cloned}
}

// Route finds the first matching processor and delegates processing to it.
func (r Router) Route(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	for _, processor := range r.processors {
		if !processor.CanProcess(sub) {
			continue
		}
		return processor.Process(sub)
	}
	return inputlifecycletype.ProcessResult{}, ErrNoProcessorMatched
}
