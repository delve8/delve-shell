package slashflow

import (
	"testing"

	"delve-shell/internal/slash/view"
)

func TestEvaluateMainEnter_NoSlash(t *testing.T) {
	got := EvaluateMainEnter("hello", EnterInput{})
	if got != OutcomeNone {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_SessionNone(t *testing.T) {
	got := EvaluateMainEnter("/history", EnterInput{
		HasSlashPrefix:      true,
		Selected:            slashview.Option{Cmd: "No sessions available."},
		VisibleOptionCount:  1,
		IsSessionNoneOption: true,
	})
	if got != OutcomeShowSessionNone {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_ResolveSelected(t *testing.T) {
	got := EvaluateMainEnter("/he", EnterInput{
		HasSlashPrefix:     true,
		Selected:           slashview.Option{Cmd: "/help"},
		VisibleOptionCount: 1,
	})
	if got != OutcomeResolveSelected {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_Unknown(t *testing.T) {
	got := EvaluateMainEnter("/zzz", EnterInput{
		HasSlashPrefix: true,
		Selected:       slashview.Option{Cmd: "/help"},
	})
	if got != OutcomeUnknownSlash {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_DelRemoteNone(t *testing.T) {
	got := EvaluateMainEnter("/config del-remote", EnterInput{
		HasSlashPrefix:        true,
		Selected:              slashview.Option{Cmd: "No hosts."},
		VisibleOptionCount:    1,
		IsDelRemoteNoneOption: true,
	})
	if got != OutcomeShowDelRemoteNone {
		t.Fatalf("unexpected outcome: %v", got)
	}
}
