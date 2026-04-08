package inputpreflight

import (
	"strings"

	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/slash/flow"
	"delve-shell/internal/slash/view"
)

// EnterPlanKind is the action selected for a slash Enter key press during pre-input handling.
type EnterPlanKind string

const (
	EnterPlanNone     EnterPlanKind = "none"
	EnterPlanFillOnly EnterPlanKind = "fill_only"
	EnterPlanSubmit   EnterPlanKind = "submit"
)

// EnterPlan is the pre-input result for a slash Enter key press.
type EnterPlan struct {
	Kind       EnterPlanKind
	FillValue  string
	Submission inputlifecycletype.InputSubmission
}

// PlanSlashEnter preserves the current slash early-enter behavior:
// fill-only rows stay in the input; otherwise a slash submission is formed.
func PlanSlashEnter(inputVal string, selected slashview.Option, hasSelected bool, selectedIndex int) EnterPlan {
	trimmed := strings.TrimSpace(inputVal)
	if trimmed == "" {
		return EnterPlan{Kind: EnterPlanNone}
	}

	result := slashflow.EvaluateSlashEnter(inputVal, trimmed, selected, hasSelected)
	switch result.Action {
	case slashflow.EnterKeyFillOnly:
		return EnterPlan{
			Kind:      EnterPlanFillOnly,
			FillValue: result.Fill,
		}
	default:
		return EnterPlan{
			Kind: EnterPlanSubmit,
			Submission: inputlifecycletype.InputSubmission{
				Kind:          inputlifecycletype.SubmissionSlash,
				Source:        inputlifecycletype.SourceSlashEarlyEnter,
				RawText:       trimmed,
				InputLine:     inputVal,
				SelectedIndex: selectedIndex,
			},
		}
	}
}
