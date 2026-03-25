package inputlifecycle

import "delve-shell/internal/inputlifecycletype"

// Engine combines pre-input submission creation with routed processing.
type Engine struct {
	preflight inputlifecycletype.PreInputEngine
	router    inputlifecycletype.SubmissionRouter
}

// NewEngine creates a lifecycle engine from a pre-input engine and router.
func NewEngine(preflight inputlifecycletype.PreInputEngine, router inputlifecycletype.SubmissionRouter) Engine {
	return Engine{
		preflight: preflight,
		router:    router,
	}
}

// SubmitEnter normalizes the current input buffer and routes it through the unified lifecycle.
func (e Engine) SubmitEnter(current string, selectedIndex int) (inputlifecycletype.ProcessResult, bool, error) {
	if e.preflight == nil || e.router == nil {
		return inputlifecycletype.ProcessResult{}, false, nil
	}
	sub, ok := e.preflight.OnEnter(current, selectedIndex)
	if !ok {
		return inputlifecycletype.ProcessResult{}, false, nil
	}
	return e.RouteSubmission(sub)
}

// RouteSubmission routes a pre-built submission through the unified lifecycle.
func (e Engine) RouteSubmission(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, bool, error) {
	if e.router == nil {
		return inputlifecycletype.ProcessResult{}, false, nil
	}
	res, err := e.router.Route(sub)
	return res, true, err
}

// SubmitControl routes an explicit control signal through the unified lifecycle.
func (e Engine) SubmitControl(signal inputlifecycletype.ControlSignal, source inputlifecycletype.SubmissionSource) (inputlifecycletype.ProcessResult, error) {
	if e.router == nil {
		return inputlifecycletype.ProcessResult{}, nil
	}
	return e.router.Route(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionControl,
		Source:        source,
		SelectedIndex: -1,
		ControlSignal: signal,
	})
}
