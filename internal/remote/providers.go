package remote

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func registerProviders() {
	ui.RegisterSlashOptionsProvider(remoteSlashOptionsProvider)

	ui.RegisterTitleBarFragmentProvider(func(m ui.Model) (string, bool) {
		if !m.Remote.Active {
			return "", false
		}
		if lbl := m.Remote.Label; lbl != "" {
			return "Remote " + lbl, true
		}
		return "Remote", true
	})

	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		return handleRemoteOverlayKey(m, key, msg)
	})

	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildRemoteOverlayContent(m)
	})

	ui.RegisterMessageProvider(remoteMessageProvider)

	ui.RegisterOverlayCloseHook(func(m ui.Model) ui.Model {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
		return m
	})
}
