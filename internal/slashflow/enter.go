package slashflow

import (
	"strings"

	"delve-shell/internal/slashview"
)

type Outcome int

const (
	OutcomeNone Outcome = iota
	OutcomeShowSessionNone
	OutcomeShowDelRemoteNone
	OutcomeResolveSelected
	OutcomeUnknownSlash
)

type EnterInput struct {
	HasSlashPrefix        bool
	SelectedCmd           string
	VisibleOptionCount    int
	IsSessionNoneOption   bool
	IsDelRemoteNoneOption bool
}

// EvaluateMainEnter determines slash-enter outcome after exact/prefix dispatch misses.
func EvaluateMainEnter(input string, in EnterInput) Outcome {
	if !in.HasSlashPrefix {
		return OutcomeNone
	}
	if in.SelectedCmd == "" {
		return OutcomeUnknownSlash
	}
	if in.VisibleOptionCount == 1 && in.IsSessionNoneOption {
		return OutcomeShowSessionNone
	}
	if in.VisibleOptionCount == 1 && in.IsDelRemoteNoneOption &&
		strings.HasPrefix(strings.TrimSpace(input), "/config del-remote") {
		return OutcomeShowDelRemoteNone
	}
	if slashview.ShouldResolveSelected(in.SelectedCmd, input) {
		return OutcomeResolveSelected
	}
	return OutcomeUnknownSlash
}
