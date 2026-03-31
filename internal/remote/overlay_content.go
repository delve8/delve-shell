package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

// pathCompletionFixedRows is the fixed height (in lines) for path completion lists in remote overlays.
const pathCompletionFixedRows = 4

// pathCompletionWindow returns up to maxRows entries from cands, scrolled so index stays visible.
func pathCompletionWindow(cands []string, index int, maxRows int) (start int, window []string) {
	if len(cands) == 0 || maxRows <= 0 {
		return 0, nil
	}
	if len(cands) <= maxRows {
		return 0, cands
	}
	if index < 0 {
		index = 0
	}
	if index >= len(cands) {
		index = len(cands) - 1
	}
	start = index - maxRows/2
	if start < 0 {
		start = 0
	}
	if start+maxRows > len(cands) {
		start = len(cands) - maxRows
	}
	return start, cands[start : start+maxRows]
}

// appendPathCompletionBlock renders one title row (or a blank row when showTitle is false) plus
// pathCompletionFixedRows list rows, so total height stays constant when toggling the title.
func appendPathCompletionBlock(b *strings.Builder, showTitle bool, cands []string, selectedIndex int) {
	// Caller is responsible for spacing before this block (e.g. one "\n\n" after the text input).
	if showTitle {
		b.WriteString(ui.RenderOverlayPicklistHintLine())
	} else {
		b.WriteString("\n")
	}
	if len(cands) == 0 {
		// Keep the same row count as when candidates exist so the overlay height does not jump.
		for i := 0; i < pathCompletionFixedRows; i++ {
			b.WriteString(ui.SuggestStyleRender("  ") + "\n")
		}
		return
	}
	start, win := pathCompletionWindow(cands, selectedIndex, pathCompletionFixedRows)
	for len(win) < pathCompletionFixedRows {
		win = append(win, "")
	}
	for i := 0; i < pathCompletionFixedRows; i++ {
		abs := start + i
		text := strings.TrimRight(win[i], "\n")
		line := "  " + text
		if win[i] == "" {
			line = "  "
		}
		highlight := abs >= 0 && abs < len(cands) && abs == selectedIndex
		if highlight {
			b.WriteString(ui.SuggestHiRender(line) + "\n")
		} else {
			b.WriteString(ui.SuggestStyleRender(line) + "\n")
		}
	}
}

func buildRemoteOverlayContent(m ui.Model) (string, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	if state.AddRemote.Active {
		var b strings.Builder
		if state.AddRemote.Connecting {
			b.WriteString("Add remote\n\n")
			b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
			b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
			return b.String(), true
		}

		if state.AddRemote.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.AddRemote.Error) + "\n\n")
			if state.AddRemote.OfferOverwrite {
				b.WriteString("Press y to overwrite, or change host/username and try again.\n\n")
			}
		}
		b.WriteString("Host (address or host:port):\n")
		b.WriteString(state.AddRemote.HostInput.View())
		b.WriteString("\n\n")
		b.WriteString("Username:\n")
		b.WriteString(state.AddRemote.UserInput.View())
		b.WriteString("\n\n")
		b.WriteString("Key path (optional):\n")
		b.WriteString(state.AddRemote.KeyInput.View())
		b.WriteString("\n\n")
		// Fixed total height: title line only when Key path is focused; blank line otherwise; 4 rows below.
		keyFocused := state.AddRemote.FieldIndex == 2
		cands := pcState.Candidates
		idx := pcState.Index
		if !keyFocused {
			cands = nil
			idx = 0
		}
		appendPathCompletionBlock(&b, keyFocused, cands, idx)
		b.WriteString("\n")
		saveLabel := "[ ]"
		if state.AddRemote.Save {
			saveLabel = "[X]"
		}
		saveLine := saveLabel + " Save as remote (Space to toggle)"
		if state.AddRemote.FieldIndex == 3 {
			b.WriteString(ui.SuggestHiRender(saveLine) + "\n")
		} else {
			b.WriteString(ui.SuggestStyleRender(saveLine) + "\n")
		}
		if state.AddRemote.Save {
			b.WriteString("\n")
			b.WriteString("Name (optional):\n")
			b.WriteString(state.AddRemote.NameInput.View())
		}
		b.WriteString("\n\n")
		b.WriteString(ui.RenderOverlayFormFooterHint())
		return b.String(), true
	}

	switch state.RemoteAuth.Step {
	case "hostkey":
		return buildRemoteAuthHostKeyContent(state), true
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
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEnterContinueEsc))
	return b.String()
}

func buildRemoteAuthChoiceContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("Choose authentication method:\n")
	b.WriteString("  1. Password\n")
	b.WriteString("  2. Key file (identity file)\n\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlay12SelectEsc))
	return b.String()
}

func buildRemoteAuthPasswordContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH password for " + remoteAuthHostLabel(state) + "\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender("Connecting...") + "\n\n")
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
	} else {
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEnterSubmitEsc))
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
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
	} else {
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEnterSubmitEsc))
	}
	b.WriteString("\n\n")
	b.WriteString(state.RemoteAuth.Input.View())
	b.WriteString("\n\n")
	appendPathCompletionBlock(&b, true, pcState.Candidates, pcState.Index)
	return b.String()
}

func buildRemoteAuthAutoIdentityContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString("SSH auth for " + remoteAuthHostLabel(state) + "\n\n")
	b.WriteString(ui.SuggestStyleRender("Connecting with configured SSH key...") + "\n\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
	return b.String()
}

func buildRemoteAuthHostKeyContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	host := state.RemoteAuth.HostKeyHost
	if strings.TrimSpace(host) == "" {
		host = remoteAuthHostLabel(state)
	}
	b.WriteString("Host key verification\n\n")
	b.WriteString("Target: " + host + "\n")
	if strings.TrimSpace(state.RemoteAuth.HostKeyFP) != "" {
		b.WriteString("Fingerprint: " + state.RemoteAuth.HostKeyFP + "\n")
	}
	b.WriteString("\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender("Updating known_hosts and reconnecting...") + "\n\n")
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
		return b.String()
	}
	b.WriteString("1. Accept and update known_hosts\n")
	b.WriteString("2. Reject and abort\n\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlay12SelectEsc))
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
