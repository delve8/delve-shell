package chatproc

import (
	"errors"

	"delve-shell/internal/input/lifecycletype"
)

// ErrChatExecutorMissing is returned when chat processing is requested without an executor.
var ErrChatExecutorMissing = errors.New("chat processor: executor is nil")

// Executor adapts the current chat/AI runtime into the unified lifecycle.
type Executor interface {
	ExecuteChat(inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error)
}

// Processor handles chat submissions.
type Processor struct {
	executor Executor
}

// New creates a chat processor that delegates chat execution to an adapter.
func New(executor Executor) Processor {
	return Processor{executor: executor}
}

// CanProcess reports whether the submission belongs to the chat branch.
func (p Processor) CanProcess(sub inputlifecycletype.InputSubmission) bool {
	return sub.Kind == inputlifecycletype.SubmissionChat
}

// Process delegates chat execution to the configured executor.
func (p Processor) Process(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	if p.executor == nil {
		return inputlifecycletype.ProcessResult{}, ErrChatExecutorMissing
	}
	return p.executor.ExecuteChat(sub)
}
