package ui

// ApplyOverlayCloseFeatureResets clears feature-owned overlay fields when any overlay is dismissed
// (Esc or programmatic close). Keep in sync with overlay key handlers and overlay open paths.
func ApplyOverlayCloseFeatureResets(m Model) Model {
	// Remote + remote auth (internal/remote).
	m.AddRemote.Active = false
	m.AddRemote.Connecting = false
	m.AddRemote.Error = ""
	m.AddRemote.OfferOverwrite = false
	m.RemoteAuth.Connecting = false
	m.RemoteAuth.Step = ""
	m.RemoteAuth.Target = ""
	m.RemoteAuth.Error = ""
	m.RemoteAuth.Username = ""

	// Skill overlays (internal/skill).
	m.AddSkill.Active = false
	m.AddSkill.Error = ""
	m.UpdateSkill.Active = false
	m.UpdateSkill.Error = ""

	// Config LLM overlay (internal/configllm).
	m.ConfigLLM.Active = false
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""

	return m
}

func init() {
	RegisterOverlayCloseHook(ApplyOverlayCloseFeatureResets)
}
