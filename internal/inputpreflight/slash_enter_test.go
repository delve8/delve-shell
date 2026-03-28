package inputpreflight

import (
	"testing"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/slashview"
)

func TestPlanSlashEnter(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		got := PlanSlashEnter("   ", slashview.Option{}, false, 0)
		if got.Kind != EnterPlanNone {
			t.Fatalf("Kind=%q want none", got.Kind)
		}
	})

	t.Run("fill only", func(t *testing.T) {
		got := PlanSlashEnter("/c", slashview.Option{Cmd: "/config"}, true, 0)
		if got.Kind != EnterPlanFillOnly {
			t.Fatalf("Kind=%q want fill_only", got.Kind)
		}
		if got.FillValue != "/config " {
			t.Fatalf("FillValue=%q want /config ", got.FillValue)
		}
	})

	t.Run("submit slash", func(t *testing.T) {
		got := PlanSlashEnter("/help", slashview.Option{Cmd: "/help"}, true, 2)
		if got.Kind != EnterPlanSubmit {
			t.Fatalf("Kind=%q want submit", got.Kind)
		}
		if got.Submission.Kind != inputlifecycletype.SubmissionSlash {
			t.Fatalf("Submission.Kind=%q want slash", got.Submission.Kind)
		}
		if got.Submission.Source != inputlifecycletype.SourceSlashEarlyEnter {
			t.Fatalf("Submission.Source=%q want slash_early_enter", got.Submission.Source)
		}
		if got.Submission.RawText != "/help" {
			t.Fatalf("RawText=%q want /help", got.Submission.RawText)
		}
		if got.Submission.InputLine != "/help" {
			t.Fatalf("InputLine=%q want /help", got.Submission.InputLine)
		}
		if got.Submission.SelectedIndex != 2 {
			t.Fatalf("SelectedIndex=%d want 2", got.Submission.SelectedIndex)
		}
	})

	t.Run("quit becomes control submission", func(t *testing.T) {
		got := PlanSlashEnter("/quit", slashview.Option{Cmd: "/quit"}, true, 5)
		if got.Kind != EnterPlanSubmit {
			t.Fatalf("Kind=%q want submit", got.Kind)
		}
		if got.Submission.Kind != inputlifecycletype.SubmissionControl {
			t.Fatalf("Submission.Kind=%q want control", got.Submission.Kind)
		}
		if got.Submission.ControlSignal != inputlifecycletype.ControlSignalQuit {
			t.Fatalf("ControlSignal=%q want quit", got.Submission.ControlSignal)
		}
	})
}
