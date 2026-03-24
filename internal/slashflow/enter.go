package slashflow

import "delve-shell/internal/slashview"

type Outcome int

const (
	OutcomeNone Outcome = iota
	OutcomeSwitchSession
	OutcomeShowSessionNone
	OutcomeResolveSelected
	OutcomeUnknownSlash
)

type EnterInput struct {
	HasSlashPrefix      bool
	SelectedPath        string
	SelectedCmd         string
	VisibleOptionCount  int
	IsSessionNoneOption bool
}

// EvaluateMainEnter determines slash-enter outcome after exact/prefix dispatch misses.
func EvaluateMainEnter(input string, in EnterInput) Outcome {
	if !in.HasSlashPrefix {
		return OutcomeNone
	}
	if in.SelectedPath != "" {
		return OutcomeSwitchSession
	}
	if in.SelectedCmd == "" {
		return OutcomeUnknownSlash
	}
	if in.VisibleOptionCount == 1 && in.IsSessionNoneOption {
		return OutcomeShowSessionNone
	}
	if slashview.ShouldResolveSelected(in.SelectedCmd, input) {
		return OutcomeResolveSelected
	}
	return OutcomeUnknownSlash
}
