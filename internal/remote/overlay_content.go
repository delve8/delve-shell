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

func buildRemoteOverlayContent(m *ui.Model) (string, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	if state.AddRemote.Active {
		var b strings.Builder
		if state.AddRemote.Connecting {
			b.WriteString(i18n.T(i18n.KeyAddRemoteScreenTitle) + "\n\n")
			b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyAddRemoteConnecting)) + "\n\n")
			b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
			return b.String(), true
		}

		if state.AddRemote.Error != "" {
			b.WriteString(ui.ErrStyleRender(state.AddRemote.Error) + "\n\n")
			if state.AddRemote.OfferOverwrite {
				b.WriteString(i18n.T(i18n.KeyAddRemoteOverwriteHint) + "\n\n")
			}
		}
		b.WriteString(i18n.T(i18n.KeyAddRemoteHostLabel) + "\n")
		b.WriteString(state.AddRemote.HostInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(i18n.KeyAddRemoteUserLabel) + "\n")
		b.WriteString(state.AddRemote.UserInput.View())
		b.WriteString("\n\n")
		b.WriteString(i18n.T(i18n.KeyAddRemoteKeyLabel) + "\n")
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
		saveLine := saveLabel + " " + i18n.T(i18n.KeyAddRemoteSaveLabel)
		if state.AddRemote.FieldIndex == 3 {
			b.WriteString(ui.SuggestHiRender(saveLine) + "\n")
		} else {
			b.WriteString(ui.SuggestStyleRender(saveLine) + "\n")
		}
		if state.AddRemote.Save {
			b.WriteString("\n")
			b.WriteString(i18n.T(i18n.KeyAddRemoteNameLabel) + "\n")
			b.WriteString(state.AddRemote.NameInput.View())
		}
		b.WriteString("\n\n")
		b.WriteString(ui.RenderOverlayFormFooterHint())
		return b.String(), true
	}

	switch state.RemoteAuth.Step {
	case AuthStepHostKey:
		return buildRemoteAuthHostKeyContent(state), true
	case AuthStepUsername:
		return buildRemoteAuthUsernameContent(state), true
	case AuthStepChoose:
		return buildRemoteAuthChoiceContent(state), true
	case AuthStepPassword:
		return buildRemoteAuthPasswordContent(state), true
	case AuthStepIdentity:
		return buildRemoteAuthIdentityContent(state, pcState), true
	case AuthStepAutoIdentity:
		return buildRemoteAuthAutoIdentityContent(state), true
	default:
		return "", false
	}
}

func buildRemoteAuthUsernameContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString(i18n.Tf(i18n.KeyRemoteAuthUsernameTitle, remoteAuthHostLabel(state)) + "\n\n")
	b.WriteString(i18n.T(i18n.KeyAddRemoteUserLabel) + "\n")
	b.WriteString(state.RemoteAuth.UsernameInput.View())
	b.WriteString("\n\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEnterContinueEsc))
	return b.String()
}

func buildRemoteAuthChoiceContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString(i18n.T(i18n.KeyRemoteAuthMethodTitle) + "\n")
	appendRemoteAuthChoiceLine(&b, 0, state.RemoteAuth.ChoiceIndex, i18n.T(i18n.KeyRemoteAuthPasswordChoice))
	appendRemoteAuthChoiceLine(&b, 1, state.RemoteAuth.ChoiceIndex, i18n.T(i18n.KeyRemoteAuthIdentityChoice))
	b.WriteString("\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayChoiceSelectEsc))
	return b.String()
}

func buildRemoteAuthPasswordContent(state remoteOverlayState) string {
	var b strings.Builder
	appendRemoteAuthError(&b, state.RemoteAuth.Error)
	b.WriteString(i18n.Tf(i18n.KeyRemoteAuthPasswordTitle, remoteAuthHostLabel(state)) + "\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyRemoteAuthConnecting)) + "\n\n")
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
	b.WriteString(i18n.Tf(i18n.KeyRemoteAuthIdentityTitle, remoteAuthHostLabel(state)) + "\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyRemoteAuthConnecting)) + "\n\n")
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
	b.WriteString(i18n.Tf(i18n.KeyRemoteAuthAutoIdentityTitle, remoteAuthHostLabel(state)) + "\n\n")
	b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyRemoteAuthConfiguredKey)) + "\n\n")
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
	b.WriteString(i18n.T(i18n.KeyRemoteAuthHostKeyTitle) + "\n\n")
	b.WriteString(i18n.Tf(i18n.KeyRemoteAuthTargetLabel, host) + "\n")
	if strings.TrimSpace(state.RemoteAuth.HostKeyFP) != "" {
		b.WriteString(i18n.Tf(i18n.KeyRemoteAuthFingerprintLabel, state.RemoteAuth.HostKeyFP) + "\n")
	}
	b.WriteString("\n")
	if state.RemoteAuth.Connecting {
		b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyRemoteAuthKnownHostsUpdate)) + "\n\n")
		b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayEscCancel))
		return b.String()
	}
	appendRemoteAuthChoiceLine(&b, 0, state.RemoteAuth.ChoiceIndex, i18n.T(i18n.KeyRemoteAuthAcceptKnownHosts))
	appendRemoteAuthChoiceLine(&b, 1, state.RemoteAuth.ChoiceIndex, i18n.T(i18n.KeyRemoteAuthRejectKnownHosts))
	b.WriteString("\n")
	b.WriteString(ui.RenderOverlayHintLine(i18n.KeyOverlayChoiceSelectEsc))
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

func appendRemoteAuthChoiceLine(b *strings.Builder, choiceIndex int, selectedIndex int, label string) {
	line := "  " + label
	if choiceIndex == selectedIndex {
		b.WriteString(ui.SuggestHiRender(line) + "\n")
		return
	}
	b.WriteString(ui.SuggestStyleRender(line) + "\n")
}
