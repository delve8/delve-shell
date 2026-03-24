package slashflow

import "testing"

func TestEvaluateMainEnter_NoSlash(t *testing.T) {
	got := EvaluateMainEnter("hello", EnterInput{})
	if got != OutcomeNone {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_SessionNone(t *testing.T) {
	got := EvaluateMainEnter("/sessions", EnterInput{
		HasSlashPrefix:      true,
		SelectedCmd:         "No sessions available.",
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
		SelectedCmd:        "/help",
		VisibleOptionCount: 1,
	})
	if got != OutcomeResolveSelected {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_Unknown(t *testing.T) {
	got := EvaluateMainEnter("/zzz", EnterInput{
		HasSlashPrefix: true,
		SelectedCmd:    "/help",
	})
	if got != OutcomeUnknownSlash {
		t.Fatalf("unexpected outcome: %v", got)
	}
}

func TestEvaluateMainEnter_DelRemoteNone(t *testing.T) {
	got := EvaluateMainEnter("/config del-remote", EnterInput{
		HasSlashPrefix:        true,
		SelectedCmd:           "No hosts.",
		VisibleOptionCount:    1,
		IsDelRemoteNoneOption: true,
	})
	if got != OutcomeShowDelRemoteNone {
		t.Fatalf("unexpected outcome: %v", got)
	}
}
