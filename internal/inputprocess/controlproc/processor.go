package controlproc

import (
	"errors"

	"delve-shell/internal/inputlifecycletype"
)

// ErrUnknownControlSignal is returned when a control submission does not carry a supported signal.
var ErrUnknownControlSignal = errors.New("control processor: unknown control signal")

// ContextProvider supplies runtime state needed to resolve stateful controls such as Esc.
type ContextProvider interface {
	ControlContext() inputlifecycletype.ControlContext
}

// Executor performs the resolved control action and returns a normalized result.
type Executor interface {
	ExecuteControl(inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error)
}

// Processor handles control submissions within the unified lifecycle.
type Processor struct {
	contexts ContextProvider
	executor Executor
}

// New creates a control processor.
func New(contexts ContextProvider, executor Executor) Processor {
	return Processor{
		contexts: contexts,
		executor: executor,
	}
}

// CanProcess reports whether the submission belongs to the control lifecycle branch.
func (p Processor) CanProcess(sub inputlifecycletype.InputSubmission) bool {
	return sub.Kind == inputlifecycletype.SubmissionControl
}

// Process resolves the requested control signal into a concrete action and executes it.
func (p Processor) Process(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	action, ok := p.resolveAction(sub)
	if !ok {
		return inputlifecycletype.ProcessResult{}, ErrUnknownControlSignal
	}
	if p.executor == nil {
		return inputlifecycletype.ConsumedResult(), nil
	}
	return p.executor.ExecuteControl(action)
}

func (p Processor) resolveAction(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ControlAction, bool) {
	switch sub.ControlSignal {
	case inputlifecycletype.ControlSignalEsc:
		if p.contexts == nil {
			return "", false
		}
		return inputlifecycletype.ResolveEscAction(p.contexts.ControlContext())
	case inputlifecycletype.ControlSignalQuit:
		return inputlifecycletype.ControlQuit, true
	case inputlifecycletype.ControlSignalInterrupt:
		return inputlifecycletype.ControlInterrupt, true
	default:
		return "", false
	}
}
