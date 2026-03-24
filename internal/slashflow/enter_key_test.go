package slashflow

import (
	"testing"

	"delve-shell/internal/slashview"
)

func TestEvaluateSlashEnter_FillOnly(t *testing.T) {
	got := EvaluateSlashEnter("/r", "/r", slashview.Option{Cmd: "/run <cmd>"}, true)
	if got.Action != EnterKeyFillOnly || got.Fill != "/run " {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestEvaluateSlashEnter_ExactChosen(t *testing.T) {
	got := EvaluateSlashEnter("/help", "/help", slashview.Option{Cmd: "/help"}, true)
	if got.Action != EnterKeyDispatchExactChosen {
		t.Fatalf("unexpected result: %+v", got)
	}
}
