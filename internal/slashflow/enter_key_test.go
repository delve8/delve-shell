package slashflow

import (
	"testing"

	"delve-shell/internal/slashview"
)

func TestEvaluateSlashEnter_FillOnly(t *testing.T) {
	got := EvaluateSlashEnter("/e", "/e", slashview.Option{Cmd: "/exec <cmd>"}, true)
	if got.Action != EnterKeyFillOnly || got.Fill != "/exec " {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestEvaluateSlashEnter_ExactChosen(t *testing.T) {
	got := EvaluateSlashEnter("/help", "/help", slashview.Option{Cmd: "/help"}, true)
	if got.Action != EnterKeyDispatchExactChosen {
		t.Fatalf("unexpected result: %+v", got)
	}
}
