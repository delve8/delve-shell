package skill

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func handleSkillOverlayEvent(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	if m.Overlay.Key != OverlayFeatureKey {
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
}
