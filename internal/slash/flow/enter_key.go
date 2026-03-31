package slashflow

import "delve-shell/internal/slash/view"

type EnterKeyAction int

const (
	EnterKeyNoop EnterKeyAction = iota
	EnterKeyDispatchExactChosen
	EnterKeyFillOnly
)

type EnterKeyResult struct {
	Action EnterKeyAction
	Fill   string
}

// EvaluateSlashEnter determines action for slash-mode Enter key.
func EvaluateSlashEnter(input string, trimmed string, selected slashview.Option, hasSelected bool) EnterKeyResult {
	if trimmed == "" {
		return EnterKeyResult{Action: EnterKeyNoop}
	}
	if selected.Cmd == "" || !hasSelected {
		return EnterKeyResult{Action: EnterKeyNoop}
	}
	if slashview.ShouldFillOnly(selected, input) {
		return EnterKeyResult{
			Action: EnterKeyFillOnly,
			Fill:   slashview.ChosenToInputValue(selected),
		}
	}
	if selected.Cmd == trimmed {
		return EnterKeyResult{Action: EnterKeyDispatchExactChosen}
	}
	return EnterKeyResult{Action: EnterKeyNoop}
}
