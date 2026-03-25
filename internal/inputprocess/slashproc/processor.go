package slashproc

import (
	"errors"

	"delve-shell/internal/inputlifecycletype"
)

// ErrSlashExecutorMissing is returned when slash processing is requested without an executor.
var ErrSlashExecutorMissing = errors.New("slash processor: executor is nil")

// ExecutionRequest is the normalized slash work item handed to an executor adapter.
type ExecutionRequest struct {
	RawText       string
	InputLine     string
	SelectedIndex int
}

// Executor adapts the current slash implementation into the unified lifecycle.
type Executor interface {
	ExecuteSlash(ExecutionRequest) (inputlifecycletype.ProcessResult, error)
}

// Processor handles slash submissions.
type Processor struct {
	executor Executor
}

// New creates a slash processor that delegates execution to an adapter.
func New(executor Executor) Processor {
	return Processor{executor: executor}
}

// CanProcess reports whether the submission belongs to the slash branch.
func (p Processor) CanProcess(sub inputlifecycletype.InputSubmission) bool {
	return sub.Kind == inputlifecycletype.SubmissionSlash
}

// Process delegates slash execution to the configured executor.
func (p Processor) Process(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	if p.executor == nil {
		return inputlifecycletype.ProcessResult{}, ErrSlashExecutorMissing
	}
	return p.executor.ExecuteSlash(ExecutionRequest{
		RawText:       sub.RawText,
		InputLine:     sub.InputLine,
		SelectedIndex: sub.SelectedIndex,
	})
}
