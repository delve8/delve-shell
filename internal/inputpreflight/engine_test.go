package inputpreflight

import (
	"testing"

	"delve-shell/internal/inputlifecycletype"
)

func TestEngineOnInputChanged(t *testing.T) {
	engine := Engine{}

	t.Run("plain input", func(t *testing.T) {
		got := engine.OnInputChanged("hello")
		if got.Mode != inputlifecycletype.PreInputModePlain {
			t.Fatalf("Mode=%q want plain", got.Mode)
		}
		if got.HasCandidates {
			t.Fatal("plain input should not expose slash candidates")
		}
	})

	t.Run("slash input", func(t *testing.T) {
		got := engine.OnInputChanged(" /access New")
		if got.Mode != inputlifecycletype.PreInputModeSlash {
			t.Fatalf("Mode=%q want slash", got.Mode)
		}
		if !got.HasCandidates {
			t.Fatal("slash input should expose slash candidates")
		}
	})
}

func TestEngineOnEnter(t *testing.T) {
	engine := Engine{}

	t.Run("empty ignored", func(t *testing.T) {
		if _, ok := engine.OnEnter("   ", 0); ok {
			t.Fatal("empty input should not produce a submission")
		}
	})

	t.Run("chat submission", func(t *testing.T) {
		got, ok := engine.OnEnter(" hello ", 3)
		if !ok {
			t.Fatal("expected chat submission")
		}
		if got.Kind != inputlifecycletype.SubmissionChat {
			t.Fatalf("Kind=%q want chat", got.Kind)
		}
		if got.RawText != "hello" {
			t.Fatalf("RawText=%q want hello", got.RawText)
		}
		if got.SelectedIndex != -1 {
			t.Fatalf("SelectedIndex=%d want -1", got.SelectedIndex)
		}
	})

	t.Run("slash submission", func(t *testing.T) {
		got, ok := engine.OnEnter(" /help ", 2)
		if !ok {
			t.Fatal("expected slash submission")
		}
		if got.Kind != inputlifecycletype.SubmissionSlash {
			t.Fatalf("Kind=%q want slash", got.Kind)
		}
		if got.RawText != "/help" {
			t.Fatalf("RawText=%q want /help", got.RawText)
		}
		if got.SelectedIndex != 2 {
			t.Fatalf("SelectedIndex=%d want 2", got.SelectedIndex)
		}
	})

	t.Run("quit control submission", func(t *testing.T) {
		got, ok := engine.OnEnter(" /q ", 7)
		if !ok {
			t.Fatal("expected control submission")
		}
		if got.Kind != inputlifecycletype.SubmissionControl {
			t.Fatalf("Kind=%q want control", got.Kind)
		}
		if got.ControlSignal != inputlifecycletype.ControlSignalQuit {
			t.Fatalf("ControlSignal=%q want quit", got.ControlSignal)
		}
	})
}
