package skill

import "delve-shell/internal/ui"

func registerOverlayCloseHook() {
	ui.RegisterOverlayCloseHook(func(m ui.Model) ui.Model {
		m.AddSkillActive = false
		m.AddSkillError = ""
		m.UpdateSkillActive = false
		m.UpdateSkillError = ""
		return m
	})
}
