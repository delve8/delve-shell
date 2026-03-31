package remote

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

func handleRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	if key == teakey.Esc {
		// Let internal/ui do overlay-close common behavior.
		return m, nil, false
	}
	if state.AddRemote.Active {
		return handleAddRemoteOverlayKey(m, key, msg)
	}
	if state.RemoteAuth.Step != "" {
		return handleRemoteAuthOverlayKey(m, key, msg)
	}
	return m, nil, false
}
