package remote

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func TestHandleAddRemoteOverlayKey_TabMovesToNextFieldWithoutCandidates(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.AddRemote.FieldIndex = 0
	state.AddRemote.HostInput = textinput.New()
	state.AddRemote.UserInput = textinput.New()
	state.AddRemote.KeyInput = textinput.New()
	state.AddRemote.NameInput = textinput.New()
	setRemoteOverlayState(state)
	pathcomplete.ResetState()
	t.Cleanup(func() {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
	})

	_, _, handled := handleAddRemoteOverlayKey(m, "tab", tea.KeyMsg{Type: tea.KeyTab})
	if !handled {
		t.Fatal("expected tab to be handled")
	}
	got := getRemoteOverlayState()
	if got.AddRemote.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.AddRemote.FieldIndex)
	}
}

func TestHandleAddRemoteOverlayKey_EnterMovesUntilLastField(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.AddRemote.FieldIndex = 0
	state.AddRemote.HostInput = textinput.New()
	state.AddRemote.UserInput = textinput.New()
	state.AddRemote.KeyInput = textinput.New()
	state.AddRemote.NameInput = textinput.New()
	setRemoteOverlayState(state)
	pathcomplete.ResetState()
	t.Cleanup(func() {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
	})

	_, _, handled := handleAddRemoteOverlayKey(m, "enter", tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got := getRemoteOverlayState()
	if got.AddRemote.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.AddRemote.FieldIndex)
	}
}

func TestHandleAddRemoteOverlayKey_SpaceOnSaveMovesToNameWhenEnabled(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.AddRemote.FieldIndex = 3
	state.AddRemote.HostInput = textinput.New()
	state.AddRemote.UserInput = textinput.New()
	state.AddRemote.KeyInput = textinput.New()
	state.AddRemote.NameInput = textinput.New()
	setRemoteOverlayState(state)
	pathcomplete.ResetState()
	t.Cleanup(func() {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
	})

	_, _, handled := handleAddRemoteOverlayKey(m, " ", tea.KeyMsg{Type: tea.KeySpace})
	if !handled {
		t.Fatal("expected space to be handled")
	}
	got := getRemoteOverlayState()
	if !got.AddRemote.Save {
		t.Fatal("expected save to be enabled")
	}
	if got.AddRemote.FieldIndex != 4 {
		t.Fatalf("field index=%d want 4", got.AddRemote.FieldIndex)
	}
}
