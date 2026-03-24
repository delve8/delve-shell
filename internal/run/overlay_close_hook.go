package run

import "delve-shell/internal/ui"

func applyOverlayCloseFeatureResets(m ui.Model) ui.Model {
	// Remote + remote auth.
	m.AddRemote.Active = false
	m.AddRemote.Connecting = false
	m.AddRemote.Error = ""
	m.AddRemote.OfferOverwrite = false
	m.RemoteAuth.Connecting = false
	m.RemoteAuth.Step = ""
	m.RemoteAuth.Target = ""
	m.RemoteAuth.Error = ""
	m.RemoteAuth.Username = ""

	// Config LLM overlay.
	m.ConfigLLM.Active = false
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""

	return m
}

func init() {
	ui.RegisterOverlayCloseHook(applyOverlayCloseFeatureResets)
}
