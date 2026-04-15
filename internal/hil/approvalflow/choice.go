package approvalflow

import (
	"unicode/utf8"

	"delve-shell/internal/teakey"
)

// normalizeChoiceKey maps bracket-wrapped single-character paste keys ("[1]") and fullwidth
// digits to ASCII 1–3 so they match ChoiceKey1..3 the same way as typed ASCII digits.
func normalizeChoiceKey(key string) string {
	// Terminals may send CR/LF as KeyRunes; main Enter must still confirm the highlighted option.
	if key == "\r" || key == "\n" {
		return teakey.Enter
	}
	if n := len(key); n >= 3 && key[0] == '[' && key[n-1] == ']' {
		key = key[1 : n-1]
	}
	r, w := utf8.DecodeRuneInString(key)
	if w != len(key) {
		return key
	}
	switch r {
	case '１':
		return ChoiceKey1
	case '２':
		return ChoiceKey2
	case '３':
		return ChoiceKey3
	default:
		return key
	}
}

type Decision int

const (
	DecisionNone Decision = iota
	DecisionApprove
	DecisionGuide
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
	key = normalizeChoiceKey(key)
	r := Result{Handled: true, ChoiceIndex: choiceIndex}
	if choiceCount > 0 {
		// Same chords as bubbles/textarea InsertNewline: plain Enter confirms, but many keyboards
		// send shift+enter / alt+enter / ctrl+j instead; those must not fall through as unknown keys.
		if key == teakey.Enter || key == teakey.ShiftEnter || key == teakey.AltEnter || key == teakey.CtrlJ {
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
			r.Decision = DecisionGuide
		case ChoiceKey3:
			r.Decision = DecisionDismiss
		}
	}
	return r
}
