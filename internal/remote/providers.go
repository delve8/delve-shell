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

	ui.RegisterOverlayFeature(ui.OverlayFeature{
		Open: func(m ui.Model, req ui.OverlayOpenRequest) (ui.Model, tea.Cmd, bool) {
			if req.Key != "remote_add" {
				return m, nil, false
			}
			return openAddRemoteOverlay(m, req.Params["save"] == "true", req.Params["connect"] == "true"), nil, true
		},
		Key: func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
			return handleRemoteOverlayKey(m, key, msg)
		},
		Message: remoteMessageProvider,
		Event:   remoteOverlayEventProvider,
		Content: func(m ui.Model) (string, bool) {
			return buildRemoteOverlayContent(m)
		},
		Close: func(m ui.Model, activeKey string) ui.Model {
			if activeKey != "remote" {
				return m
			}
			resetRemoteOverlayState()
			pathcomplete.ResetState()
			return m
		},
	})
}
