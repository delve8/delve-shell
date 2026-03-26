package skill

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func registerOverlayFeature() {
	ui.RegisterOverlayFeature(ui.OverlayFeature{
		Open: func(m ui.Model, req ui.OverlayOpenRequest) (ui.Model, tea.Cmd, bool) {
			switch req.Key {
			case "skill_add":
				return openAddSkillOverlay(m, req.Params["url"], req.Params["ref"], req.Params["path"]), nil, true
			case "skill_update":
				return openUpdateSkillOverlay(m, req.Params["name"]), nil, true
			default:
				return m, nil, false
			}
		},
		Key: func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
			if m.Overlay.Key != "skill" {
				return m, nil, false
			}
			state := getSkillOverlayState()
			if state.AddSkill.Active {
				return handleAddSkillOverlayKey(m, key, msg)
			}
			if state.UpdateSkill.Active {
				return handleUpdateSkillOverlayKey(m, key)
			}
			return m, nil, false
		},
		Event: func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
			if m.Overlay.Key != "skill" {
				return m, nil, false
			}
			state := getSkillOverlayState()
			switch t := msg.(type) {
			case AddRefsLoadedMsg:
				if state.AddSkill.Active {
					state.AddSkill.RefsFullList = t.Refs
					state.AddSkill.RefCandidates = filterByPrefix(t.Refs, state.AddSkill.RefInput.Value())
					state.AddSkill.RefIndex = 0
					setSkillOverlayState(state)
				}
				return m, nil, true
			case AddPathsLoadedMsg:
				if state.AddSkill.Active {
					state.AddSkill.PathsFullList = t.Paths
					state = updateAddSkillPathCandidates(state)
					setSkillOverlayState(state)
				}
				return m, nil, true
			default:
				return m, nil, false
			}
		},
		Content: func(m ui.Model) (string, bool) {
			return buildSkillOverlayContent(m)
		},
		Close: func(m ui.Model, activeKey string) ui.Model {
			if activeKey != "skill" {
				return m
			}
			resetSkillOverlayState()
			return m
		},
	})
}

func openAddSkillOverlay(m ui.Model, url, ref, path string) ui.Model {
	lang := "en"
	m = m.OpenOverlayFeature("skill", i18n.T(lang, i18n.KeyAddSkillTitle), "")
	state := getSkillOverlayState()
	state.AddSkill.Active = true
	state.UpdateSkill = UpdateSkillOverlayState{}
	state.AddSkill.Error = ""
	state.AddSkill.FieldIndex = 0

	state.AddSkill.URLInput = textinput.New()
	state.AddSkill.URLInput.Placeholder = "https://github.com/owner/repo or owner/repo"
	state.AddSkill.URLInput.SetValue(url)
	state.AddSkill.URLInput.Focus()

	state.AddSkill.RefInput = textinput.New()
	state.AddSkill.RefInput.Placeholder = "main"
	state.AddSkill.RefInput.SetValue(ref)
	state.AddSkill.RefInput.Blur()

	state.AddSkill.PathInput = textinput.New()
	state.AddSkill.PathInput.Placeholder = "skills/foo"
	state.AddSkill.PathInput.SetValue(path)
	state.AddSkill.PathInput.Blur()

	state.AddSkill.NameInput = textinput.New()
	state.AddSkill.NameInput.Placeholder = "local skill name"
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
	return m
}
