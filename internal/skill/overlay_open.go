package skill

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func openSkillOverlay(m *ui.Model, req ui.OverlayOpenRequest) (*ui.Model, tea.Cmd, bool) {
	switch req.Key {
	case OverlayOpenKeyAdd:
		openAddSkillOverlay(m, req.Params["url"], req.Params["ref"], req.Params["path"])
		return m, nil, true
	case OverlayOpenKeyUpdate:
		openUpdateSkillOverlay(m, req.Params["name"])
		return m, nil, true
	default:
		return m, nil, false
	}
}

func openAddSkillOverlay(m *ui.Model, url, ref, path string) {
	m.OpenOverlayFeature(OverlayFeatureKey, i18n.T(i18n.KeyAddSkillTitle), "")
	state := getSkillOverlayState()
	state.AddSkill.Active = true
	state.UpdateSkill = UpdateSkillOverlayState{}
	state.AddSkill.Error = ""
	state.AddSkill.FieldIndex = 0

	state.AddSkill.URLInput = textinput.New()
	state.AddSkill.URLInput.Placeholder = i18n.T(i18n.KeyAddSkillURLPlaceholder)
	state.AddSkill.URLInput.SetValue(url)
	state.AddSkill.URLInput.Focus()

	state.AddSkill.RefInput = textinput.New()
	state.AddSkill.RefInput.Placeholder = i18n.T(i18n.KeyAddSkillRefPlaceholder)
	state.AddSkill.RefInput.SetValue(ref)
	state.AddSkill.RefInput.Blur()

	state.AddSkill.PathInput = textinput.New()
	state.AddSkill.PathInput.Placeholder = i18n.T(i18n.KeyAddSkillPathPlaceholder)
	state.AddSkill.PathInput.SetValue(path)
	state.AddSkill.PathInput.Blur()

	state.AddSkill.NameInput = textinput.New()
	state.AddSkill.NameInput.Placeholder = i18n.T(i18n.KeyAddSkillNamePlaceholder)
	if p := strings.TrimSpace(path); p != "" {
		if idx := strings.LastIndex(p, "/"); idx >= 0 && idx < len(p)-1 {
			p = p[idx+1:]
		}
		state.AddSkill.NameInput.SetValue(p)
		state.AddSkill.NameInput.CursorEnd()
	} else {
		state.AddSkill.NameInput.SetValue("")
	}
	state.AddSkill.NameInput.Blur()

	state.AddSkill.RefsFullList = nil
	state.AddSkill.RefCandidates = nil
	state.AddSkill.RefIndex = 0
	state.AddSkill.PathsFullList = nil
	state.AddSkill.PathCandidates = nil
	state.AddSkill.PathIndex = 0
	setSkillOverlayState(state)
	pathcomplete.ResetState()
}
