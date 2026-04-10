package skill

import (
	"strings"
	"testing"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func TestBuildSkillOverlayContent_AddSkillUsesFixedPicklistHeights(t *testing.T) {
	i18n.SetLang("en")
	m := ui.NewModel(nil, nil)
	state := getSkillOverlayState()
	state.AddSkill.Active = true
	state.AddSkill.FieldIndex = 1
	state.AddSkill.RefCandidates = []string{"main", "release", "dev", "hotfix", "exp"}
	setSkillOverlayState(state)
	t.Cleanup(resetSkillOverlayState)

	content, ok := buildSkillOverlayContent(m)
	if !ok {
		t.Fatal("expected add-skill content")
	}
	if got := strings.Count(content, "Up/Down to move · Enter or Tab to apply"); got != 1 {
		t.Fatalf("picklist hint count=%d want 1\n%s", got, content)
	}
	if strings.Count(content, "\n") < skillPicklistFixedRows {
		t.Fatalf("content too short for fixed picklist block:\n%s", content)
	}
}

func TestBuildSkillOverlayContent_UpdateSkillShowsStandardFooterHint(t *testing.T) {
	i18n.SetLang("en")
	m := ui.NewModel(nil, nil)
	state := getSkillOverlayState()
	state.UpdateSkill.Active = true
	state.UpdateSkill.Name = "demo"
	state.UpdateSkill.URL = "https://example.com/repo"
	state.UpdateSkill.Refs = []string{"main", "release"}
	setSkillOverlayState(state)
	t.Cleanup(resetSkillOverlayState)

	content, ok := buildSkillOverlayContent(m)
	if !ok {
		t.Fatal("expected update-skill content")
	}
	if !strings.Contains(content, "Up/Down to move · Enter to update · Esc to cancel") {
		t.Fatalf("missing standard update footer hint:\n%s", content)
	}
}
