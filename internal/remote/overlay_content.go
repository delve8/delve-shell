package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/ui"
)

func buildRemoteOverlayContent(m ui.Model) (string, bool) {
	if m.AddRemote.Active {
		var b strings.Builder
		if m.AddRemote.Connecting {
			b.WriteString("Add remote\n\n")
			b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
			b.WriteString("Esc to cancel.")
			return b.String(), true
		}

		if m.AddRemote.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.AddRemote.Error) + "\n\n")
			if m.AddRemote.OfferOverwrite {
				b.WriteString("Press y to overwrite, or change host/username and try again.\n\n")
			}
		}
		b.WriteString("Add remote\n\n")
		b.WriteString("Host (address or host:port):\n")
		b.WriteString(m.AddRemote.HostInput.View())
		b.WriteString("\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.AddRemote.UserInput.View())
		b.WriteString("\n\n")
		b.WriteString("Name (optional):\n")
		b.WriteString(m.AddRemote.NameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Key path (optional):\n")
		b.WriteString(m.AddRemote.KeyInput.View())
		b.WriteString("\n\n")
		if m.AddRemote.FieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
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
		if m.AddRemote.Connect {
			saveLabel := "[ ]"
			if m.AddRemote.Save {
				saveLabel = "[X]"
			}
			saveLine := saveLabel + " Save as remote (Space to toggle)"
			if m.AddRemote.FieldIndex == 4 {
				b.WriteString(ui.SuggestHiRender(saveLine) + "\n\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(saveLine) + "\n\n")
			}
		}
		b.WriteString("Up/Down to move between fields, Enter to apply, Esc to cancel.")
		return b.String(), true
	}

	switch m.RemoteAuth.Step {
	case "username":
		var b strings.Builder
		if m.RemoteAuth.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuth.Error) + "\n\n")
		}
		b.WriteString("SSH auth for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n\n")
		b.WriteString("Username:\n")
		b.WriteString(m.RemoteAuth.UsernameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Press Enter to continue, Esc to cancel.")
		return b.String(), true
	case "choose":
		var b strings.Builder
		if m.RemoteAuth.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuth.Error) + "\n\n")
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
		b.WriteString(m.RemoteAuth.Input.View())
		return b.String(), true
	case "identity":
		var b strings.Builder
		b.WriteString(m.OverlayContent)
		b.WriteString("\n\n")
		b.WriteString(m.RemoteAuth.Input.View())
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
		if m.RemoteAuth.Error != "" {
			b.WriteString(ui.ErrStyleRender(m.RemoteAuth.Error) + "\n\n")
		}
		b.WriteString("SSH auth for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n\n")
		b.WriteString(ui.SuggestStyleRender("Connecting with configured SSH key...") + "\n\n")
		b.WriteString("Esc to cancel.")
		return b.String(), true
	default:
		return "", false
	}
}
