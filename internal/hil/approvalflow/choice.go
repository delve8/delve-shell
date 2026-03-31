package approvalflow

import "delve-shell/internal/teakey"

type Decision int

const (
	DecisionNone Decision = iota
	DecisionApprove
	DecisionReject
	DecisionCopy
	DecisionDismiss
	DecisionSensitiveRefuse
	DecisionSensitiveRunStore
	DecisionSensitiveRunNoStore
)

type Result struct {
	Handled       bool
	ChoiceIndex   int
	ChoiceChanged bool
	Decision      Decision
}

// Evaluate interprets a key in approval/sensitive choice mode.
func Evaluate(key string, hasPending bool, hasSensitive bool, choiceIndex int, choiceCount int) Result {
	if !hasPending && !hasSensitive {
		return Result{}
	}
	r := Result{Handled: true, ChoiceIndex: choiceIndex}
	if choiceCount > 0 {
		if key == teakey.Enter {
			key = string(rune('1' + choiceIndex))
		} else if key == teakey.Up || key == teakey.Down {
			if key == teakey.Down {
				r.ChoiceIndex = (choiceIndex + 1) % choiceCount
			} else {
				r.ChoiceIndex = (choiceIndex - 1 + choiceCount) % choiceCount
			}
			r.ChoiceChanged = true
			return r
		}
	}

	if hasSensitive {
		switch key {
		case ChoiceKey1:
			r.Decision = DecisionSensitiveRefuse
		case ChoiceKey2:
			r.Decision = DecisionSensitiveRunStore
		case ChoiceKey3:
			r.Decision = DecisionSensitiveRunNoStore
		}
		return r
	}

	if hasPending {
		switch key {
		case ChoiceKey1:
			r.Decision = DecisionApprove
		case ChoiceKey2:
			r.Decision = DecisionDismiss
		case ChoiceKey3:
			r.Decision = DecisionCopy
		}
	}
	return r
}
