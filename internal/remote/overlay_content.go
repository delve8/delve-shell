package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func buildRemoteOverlayContent(m ui.Model) (string, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	if state.AddRemote.Active {
		var b strings.Builder
		if state.AddRemote.Connecting {
			b.WriteString("Add remote\n\n")
			b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
			b.WriteString("Esc to cancel.")
			return b.String(), true
		}

		if state.AddRemote.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.AddRemote.Error) + "\n\n")
			if state.AddRemote.OfferOverwrite {
				b.WriteString("Press y to overwrite, or change host/username and try again.\n\n")
			}
		}
		b.WriteString("Add remote\n\n")
		b.WriteString("Host (address or host:port):\n")
		b.WriteString(state.AddRemote.HostInput.View())
		b.WriteString("\n\n")
		b.WriteString("Username:\n")
		b.WriteString(state.AddRemote.UserInput.View())
		b.WriteString("\n\n")
		b.WriteString("Name (optional):\n")
		b.WriteString(state.AddRemote.NameInput.View())
		b.WriteString("\n\n")
		b.WriteString("Key path (optional):\n")
		b.WriteString(state.AddRemote.KeyInput.View())
		b.WriteString("\n\n")
		if state.AddRemote.FieldIndex == 3 && len(pcState.Candidates) > 0 {
			b.WriteString("\n\n")
			b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
			for i, c := range pcState.Candidates {
				line := "  " + c
				if i == pcState.Index {
					b.WriteString(ui.SuggestHiRender(line) + "\n")
				} else {
					b.WriteString(ui.SuggestStyleRender(line) + "\n")
				}
			}
		}
		if state.AddRemote.Connect {
			saveLabel := "[ ]"
			if state.AddRemote.Save {
				saveLabel = "[X]"
			}
			saveLine := saveLabel + " Save as remote (Space to toggle)"
			if state.AddRemote.FieldIndex == 4 {
				b.WriteString(ui.SuggestHiRender(saveLine) + "\n\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(saveLine) + "\n\n")
			}
		}
		b.WriteString("Up/Down to move between fields, Enter to apply, Esc to cancel.")
		return b.String(), true
	}

	switch state.RemoteAuth.Step {
	case "username":
		return buildRemoteAuthUsernameContent(state), true
	case "choose":
		return buildRemoteAuthChoiceContent(state), true
	case "password":
		return buildRemoteAuthPasswordContent(state), true
	case "identity":
		return buildRemoteAuthIdentityContent(state, pcState), true
	case "auto_identity":
		return buildRemoteAuthAutoIdentityContent(state), true
	default:
		return "", false
	}
}

func buildRemoteAuthUsernameContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH auth for " + remoteAuthHostLabel(state) + "\n\n")
	b.WriteString("Username:\n")
	b.WriteString(state.RemoteAuth.UsernameInput.View())
	b.WriteString("\n\n")
	b.WriteString("Press Enter to continue, Esc to cancel.")
	return b.String()
}

func buildRemoteAuthChoiceContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("Choose authentication method:\n")
	b.WriteString("  1. Password\n")
	b.WriteString("  2. Key file (identity file)\n\n")
	b.WriteString("Press 1 or 2 to select, Esc to cancel.")
	return b.String()
}

func buildRemoteAuthPasswordContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH password for " + remoteAuthHostLabel(state) + "\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
		b.WriteString("Press Esc to cancel.")
	} else {
		b.WriteString("Press Enter to submit, Esc to cancel.")
	}
	b.WriteString("\n\n")
	b.WriteString(state.RemoteAuth.Input.View())
	return b.String()
}

func buildRemoteAuthIdentityContent(state remoteOverlayState, pcState pathcomplete.State) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH key file path for " + remoteAuthHostLabel(state) + "\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
		b.WriteString("Press Esc to cancel.")
	} else {
		b.WriteString("Press Enter to submit, Esc to cancel.")
	}
	b.WriteString("\n\n")
	b.WriteString(state.RemoteAuth.Input.View())
	if len(pcState.Candidates) > 0 {
		b.WriteString("\n\n")
		b.WriteString("Path completion (Up/Down select, Enter or Tab to pick):\n")
		for i, c := range pcState.Candidates {
			line := "  " + c
			if i == pcState.Index {
				b.WriteString(ui.SuggestHiRender(line) + "\n")
			} else {
				b.WriteString(ui.SuggestStyleRender(line) + "\n")
			}
		}
	}
	return b.String()
}

func buildRemoteAuthAutoIdentityContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH auth for " + remoteAuthHostLabel(state) + "\n\n")
	b.WriteString(ui.SuggestStyleRender("Connecting with configured SSH key...") + "\n\n")
	b.WriteString("Esc to cancel.")
	return b.String()
}

func appendRemoteAuthError(b *strings.Builder, errText string) {
	if errText == "" {
		return
	}
	b.WriteString(ui.ErrStyleRender(errText) + "\n\n")
}

func remoteAuthHostLabel(state remoteOverlayState) string {
	return config.HostFromTarget(state.RemoteAuth.Target)
}
