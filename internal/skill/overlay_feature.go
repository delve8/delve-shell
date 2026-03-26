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
	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		state := getSkillOverlayState()
		if state.AddSkill.Active {
			return handleAddSkillOverlayKey(m, key, msg)
		}
		if state.UpdateSkill.Active {
			return handleUpdateSkillOverlayKey(m, key)
		}
		return m, nil, false
	})

	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
		state := getSkillOverlayState()
		switch t := msg.(type) {
		case OpenAddSkillOverlayMsg:
			return openAddSkillOverlay(m, t.URL, t.Ref, t.Path), nil, true
		case OpenUpdateSkillOverlayMsg:
			return openUpdateSkillOverlay(m, t.Name), nil, true
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
	})

	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildSkillOverlayContent(m)
	})

	ui.RegisterOverlayCloseHook(func(m ui.Model) ui.Model {
		resetSkillOverlayState()
		return m
	})
}

func openAddSkillOverlay(m ui.Model, url, ref, path string) ui.Model {
	lang := "en"
	m = m.OpenOverlay(i18n.T(lang, i18n.KeyAddSkillTitle), "")
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
