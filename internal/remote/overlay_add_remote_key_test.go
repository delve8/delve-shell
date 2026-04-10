package remote

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
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

func TestHandleAddRemoteOverlayKey_OverwriteChoiceDownEnterKeepsEditing(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.AddRemote.OfferOverwrite = true
	state.AddRemote.Error = "remote target already exists: root@example.com"
	setRemoteOverlayState(state)
	t.Cleanup(resetRemoteOverlayState)

	_, _, handled := handleAddRemoteOverlayKey(m, tea.KeyDown.String(), tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatal("expected down to be handled")
	}
	got := getRemoteOverlayState()
	if got.AddRemote.ChoiceIndex != 1 {
		t.Fatalf("choice index=%d want 1", got.AddRemote.ChoiceIndex)
	}

	_, _, handled = handleAddRemoteOverlayKey(m, tea.KeyEnter.String(), tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got = getRemoteOverlayState()
	if got.AddRemote.OfferOverwrite {
		t.Fatal("expected overwrite prompt to be cleared")
	}
	if got.AddRemote.Error != "" {
		t.Fatalf("expected overwrite error to be cleared, got %q", got.AddRemote.Error)
	}
}

func TestFirstIncompleteAddRemoteField_NameRemainsOptionalWhenSaving(t *testing.T) {
	state := AddRemoteOverlayState{
		Save:      true,
		HostInput: textinput.New(),
		UserInput: textinput.New(),
		NameInput: textinput.New(),
	}
	state.HostInput.SetValue("example.com")
	state.UserInput.SetValue("alice")

	idx, missing := firstIncompleteAddRemoteField(state)
	if missing {
		t.Fatalf("unexpected missing field idx=%d", idx)
	}
}

func TestFirstIncompleteAddRemoteField_UsernameIsRequired(t *testing.T) {
	state := AddRemoteOverlayState{
		HostInput: textinput.New(),
		UserInput: textinput.New(),
	}
	state.HostInput.SetValue("example.com")

	idx, missing := firstIncompleteAddRemoteField(state)
	if !missing {
		t.Fatal("expected missing username")
	}
	if idx != 1 {
		t.Fatalf("missing field idx=%d want 1", idx)
	}
}

func TestRememberLastAddRemoteIdentityFile_IgnoresEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	rememberLastAddRemoteIdentityFile("")
	got, err := config.LoadLastIdentityFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("last identity file=%q want empty", got)
	}
}

func TestRememberLastAddRemoteIdentityFile_WritesConfigMemory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	rememberLastAddRemoteIdentityFile(" ~/.ssh/id_ed25519 ")
	got, err := config.LoadLastIdentityFile()
	if err != nil {
		t.Fatal(err)
	}
	if got != "~/.ssh/id_ed25519" {
		t.Fatalf("last identity file=%q want %q", got, "~/.ssh/id_ed25519")
	}
}

func TestRememberLastAddRemoteUsername_IgnoresEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	rememberLastAddRemoteUsername("")
	got, err := config.LoadLastUsername()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("last username=%q want empty", got)
	}
}

func TestRememberLastAddRemoteUsername_WritesConfigMemory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	defer t.Setenv("DELVE_SHELL_ROOT", "")

	rememberLastAddRemoteUsername(" alice ")
	got, err := config.LoadLastUsername()
	if err != nil {
		t.Fatal(err)
	}
	if got != "alice" {
		t.Fatalf("last username=%q want %q", got, "alice")
	}
}
