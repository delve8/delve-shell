package inputoutput

import (
	"testing"

	"delve-shell/internal/input/lifecycletype"
	tea "github.com/charmbracelet/bubbletea"
)

func TestApplyResultWaitingForAI(t *testing.T) {
	patch, cmd := ApplyResult(inputlifecycletype.ProcessResult{
		WaitingForAI: true,
	})
	if patch.WaitingForAI == nil || !*patch.WaitingForAI {
		t.Fatal("expected WaitingForAI patch to be true")
	}
	if cmd != nil {
		t.Fatal("did not expect quit command")
	}
}

func TestApplyResultQuit(t *testing.T) {
	patch, cmd := ApplyResult(inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputQuit,
	}))
	if !patch.Quit {
		t.Fatal("expected quit patch")
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit command")
	}
}

var _ = tea.Quit
