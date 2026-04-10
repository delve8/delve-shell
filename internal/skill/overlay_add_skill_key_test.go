package skill

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func TestHandleAddSkillOverlayKey_TabMovesToNextFieldWithoutCandidates(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getSkillOverlayState()
	state.AddSkill.Active = true
	state.AddSkill.FieldIndex = 0
	state.AddSkill.URLInput = textinput.New()
	state.AddSkill.RefInput = textinput.New()
	state.AddSkill.PathInput = textinput.New()
	state.AddSkill.NameInput = textinput.New()
	setSkillOverlayState(state)
	t.Cleanup(resetSkillOverlayState)

	_, _, handled := handleAddSkillOverlayKey(m, "tab", tea.KeyMsg{Type: tea.KeyTab})
	if !handled {
		t.Fatal("expected tab to be handled")
	}
	got := getSkillOverlayState()
	if got.AddSkill.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.AddSkill.FieldIndex)
	}
}

func TestHandleAddSkillOverlayKey_EnterMovesUntilLastField(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getSkillOverlayState()
	state.AddSkill.Active = true
	state.AddSkill.FieldIndex = 0
	state.AddSkill.URLInput = textinput.New()
	state.AddSkill.RefInput = textinput.New()
	state.AddSkill.PathInput = textinput.New()
	state.AddSkill.NameInput = textinput.New()
	setSkillOverlayState(state)
	t.Cleanup(resetSkillOverlayState)

	_, _, handled := handleAddSkillOverlayKey(m, "enter", tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got := getSkillOverlayState()
	if got.AddSkill.FieldIndex != 1 {
		t.Fatalf("field index=%d want 1", got.AddSkill.FieldIndex)
	}
}
