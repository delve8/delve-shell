package configllm

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func TestHandleOverlayKey_TabMovesToNextField(t *testing.T) {
	m := ui.NewModel(nil, nil)
	st := overlayState{
		Active:           true,
		FieldIndex:       0,
		BaseURLInput:     textinput.New(),
		ApiKeyInput:      textinput.New(),
		ModelInput:       textinput.New(),
		MaxMessagesInput: textinput.New(),
		MaxCharsInput:    textinput.New(),
	}
	setOverlayState(st)
	t.Cleanup(ResetOnOverlayClose)

	_, _, handled := handleOverlayKey(m, "tab", tea.KeyMsg{Type: tea.KeyTab})
	if !handled {
		t.Fatal("expected tab to be handled")
	}
	got := getOverlayState()
	if got.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.FieldIndex)
	}
}

func TestHandleOverlayKey_EnterMovesUntilLastField(t *testing.T) {
	m := ui.NewModel(nil, nil)
	st := overlayState{
		Active:           true,
		FieldIndex:       0,
		BaseURLInput:     textinput.New(),
		ApiKeyInput:      textinput.New(),
		ModelInput:       textinput.New(),
		MaxMessagesInput: textinput.New(),
		MaxCharsInput:    textinput.New(),
	}
	setOverlayState(st)
	t.Cleanup(ResetOnOverlayClose)

	_, _, handled := handleOverlayKey(m, "enter", tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got := getOverlayState()
	if got.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.FieldIndex)
	}
}
