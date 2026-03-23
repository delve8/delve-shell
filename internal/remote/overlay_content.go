package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/ui"
)

func buildRemoteOverlayContent(m ui.Model) (string, bool) {
	if m.AddRemoteActive {
		var b strings.Builder
		if m.AddRemoteConnecting {
			b.WriteString("Add remote\n\n")
			b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
			b.WriteString("Esc to cancel.")
			return b.String(), true
		}

		if m.AddRemoteError != "" {
			b.WriteString(ui.ErrStyleRender(m.AddRemoteError) + "\n\n")
			if m.AddRemoteOfferOverwrite {
				b.WriteString("Press y to overwrite, or change host/username and try again.\n\n")
			}
		}
		b.WriteString("Add remote\n\n")
		b.WriteString("Host (address or host:port):\n")
		b.WriteString(m.AddRemoteHostInput.View())
		b.WriteString("\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.AddRemoteUserInput.View())
		b.WriteString("\n\n")
		b.WriteString("Name (optional):\n")
		b.WriteString(m.AddRemoteNameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Key path (optional):\n")
		b.WriteString(m.AddRemoteKeyInput.View())
		b.WriteString("\n\n")
		if m.AddRemoteFieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
			b.WriteString("\n\n")
			b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
			for i, c := range m.PathCompletionCandidates {
				line := "  " + c
				if i == m.PathCompletionIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		if m.AddRemoteConnect {
			saveLabel := "[ ]"
			if m.AddRemoteSave {
				saveLabel = "[X]"
			}
			saveLine := saveLabel + " Save as remote (Space to toggle)"
			if m.AddRemoteFieldIndex == 4 {
				b.WriteString(ui.SuggestHiRender(saveLine) + "\n\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(saveLine) + "\n\n")
			}
		}
		b.WriteString("Up/Down to move between fields, Enter to apply, Esc to cancel.")
		return b.String(), true
	}

	switch m.RemoteAuthStep {
	case "username":
		var b strings.Builder
		if m.RemoteAuthError != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuthError) + "\n\n")
		}
		b.WriteString("SSH auth for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.RemoteAuthUsernameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Press Enter to continue, Esc to cancel.")
		return b.String(), true
	case "choose":
		var b strings.Builder
		if m.RemoteAuthError != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuthError) + "\n\n")
		}
		b.WriteString("Choose authentication method:\n")
		b.WriteString("  1. Password\n")
		b.WriteString("  2. Key file (identity file)\n\n")
		b.WriteString("Press 1 or 2 to select, Esc to cancel.")
		return b.String(), true
	case "password":
		var b strings.Builder
		b.WriteString(m.OverlayContent)
		b.WriteString("\n\n")
		b.WriteString(m.RemoteAuthInput.View())
		return b.String(), true
	case "identity":
		var b strings.Builder
		b.WriteString(m.OverlayContent)
		b.WriteString("\n\n")
		b.WriteString(m.RemoteAuthInput.View())
		if len(m.PathCompletionCandidates) > 0 {
			b.WriteString("\n\n")
			b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
			for i, c := range m.PathCompletionCandidates {
				line := "  " + c
				if i == m.PathCompletionIndex {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		return b.String(), true
	case "auto_identity":
		var b strings.Builder
		if m.RemoteAuthError != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuthError) + "\n\n")
		}
		b.WriteString("SSH auth for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n\n")
		b.WriteString(ui.SuggestStyleRender("Connecting with configured SSH key...") + "\n\n")
		b.WriteString("Esc to cancel.")
		return b.String(), true
	default:
		return "", false
	}
}
