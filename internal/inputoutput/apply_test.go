package inputoutput

import (
	"testing"

	"delve-shell/internal/inputlifecycletype"
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

func TestApplyResultMessage(t *testing.T) {
	msg := struct{ Value string }{Value: "slash"}
	_, cmd := ApplyResult(inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:    inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{Value: msg},
	}))
	if cmd == nil {
		t.Fatal("expected deferred message command")
	}
	got := cmd()
	if got == nil {
		t.Fatal("expected deferred message")
	}
	if typed, ok := got.(struct{ Value string }); !ok || typed.Value != "slash" {
		t.Fatalf("unexpected deferred message: %#v", got)
	}
	_ = tea.Quit
}
