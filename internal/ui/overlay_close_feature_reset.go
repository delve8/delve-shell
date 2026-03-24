package ui

// ApplyOverlayCloseFeatureResets clears feature-owned overlay fields when any overlay is dismissed
// (Esc or programmatic close). Keep in sync with overlay key handlers and overlay open paths.
func ApplyOverlayCloseFeatureResets(m Model) Model {
	// Remote + remote auth (internal/remote).
	m.AddRemoteActive = false
	m.AddRemoteConnecting = false
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.RemoteAuthConnecting = false
	m.RemoteAuthStep = ""
	m.RemoteAuthTarget = ""
	m.RemoteAuthError = ""
	m.RemoteAuthUsername = ""

	// Skill overlays (internal/skill).
	m.AddSkillActive = false
	m.AddSkillError = ""
	m.UpdateSkillActive = false
	m.UpdateSkillError = ""

	// Config LLM overlay (internal/configllm).
	m.ConfigLLMActive = false
	m.ConfigLLMChecking = false
	m.ConfigLLMError = ""

	return m
}

func init() {
	RegisterOverlayCloseHook(ApplyOverlayCloseFeatureResets)
}
