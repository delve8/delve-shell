package inputpreflight

import (
	"strings"

	"delve-shell/internal/input/lifecycletype"
)

// Engine is the first concrete PreInputEngine for the unified input lifecycle.
// It only classifies current input state and builds normalized submissions.
type Engine struct{}

// OnInputChanged summarizes the current pre-submit input mode.
func (Engine) OnInputChanged(current string) inputlifecycletype.PreInputState {
	trimmed := strings.TrimSpace(current)
	mode := inputlifecycletype.PreInputModePlain
	hasCandidates := false
	if strings.HasPrefix(trimmed, "/") {
		mode = inputlifecycletype.PreInputModeSlash
		hasCandidates = trimmed != ""
	}
	return inputlifecycletype.PreInputState{
		Mode:          mode,
		InputValue:    current,
		SelectedIndex: -1,
		HasCandidates: hasCandidates,
	}
}

// OnEnter converts the current input buffer into a normalized submission.
func (Engine) OnEnter(current string, selectedIndex int) (inputlifecycletype.InputSubmission, bool) {
	trimmed := strings.TrimSpace(current)
	if trimmed == "" {
		return inputlifecycletype.InputSubmission{}, false
	}

	submission := inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionChat,
		Source:        inputlifecycletype.SourceMainEnter,
		RawText:       trimmed,
		InputLine:     current,
		SelectedIndex: -1,
	}
	if strings.HasPrefix(trimmed, "/") {
		submission.Kind = inputlifecycletype.SubmissionSlash
		submission.SelectedIndex = selectedIndex
	}
	return submission, true
}
