package approvalflow

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
		if key == "enter" {
			key = string(rune('1' + choiceIndex))
		} else if key == "up" || key == "down" {
			if key == "down" {
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
		case "1":
			r.Decision = DecisionSensitiveRefuse
		case "2":
			r.Decision = DecisionSensitiveRunStore
		case "3":
			r.Decision = DecisionSensitiveRunNoStore
		}
		return r
	}

	if hasPending {
		switch key {
		case "1":
			r.Decision = DecisionApprove
		case "2":
			r.Decision = DecisionCopy
		case "3":
			r.Decision = DecisionDismiss
		}
	}
	return r
}
