package skill

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func registerOverlayFeature() {
	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: OverlayFeatureKey,
		Open:  openSkillOverlay,
		Key: func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
			if m.Overlay.Key != OverlayFeatureKey {
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
		Event: handleSkillOverlayEvent,
		Content: func(m ui.Model) (string, bool) {
			return buildSkillOverlayContent(m)
		},
		Close: func(m ui.Model, activeKey string) ui.Model {
			if activeKey != OverlayFeatureKey {
				return m
			}
			resetSkillOverlayState()
			return m
		},
	})
}
