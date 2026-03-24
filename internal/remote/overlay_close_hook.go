package remote

import "delve-shell/internal/ui"

func registerOverlayCloseHook() {
	ui.RegisterOverlayCloseHook(func(m ui.Model) ui.Model {
		m.AddRemoteActive = false
		m.AddRemoteConnecting = false
		m.AddRemoteError = ""
		m.AddRemoteOfferOverwrite = false
		m.RemoteAuthConnecting = false
		m.RemoteAuthStep = ""
		m.RemoteAuthTarget = ""
		m.RemoteAuthError = ""
		m.RemoteAuthUsername = ""
		return m
	})
}
