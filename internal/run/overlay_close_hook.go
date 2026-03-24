package run

import "delve-shell/internal/ui"

func applyOverlayCloseFeatureResets(m ui.Model) ui.Model {
	// Config LLM overlay.
	m.ConfigLLM.Active = false
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""

	return m
}

func init() {
	ui.RegisterOverlayCloseHook(applyOverlayCloseFeatureResets)
}
