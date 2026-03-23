package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleRemoteConnectDoneMsg(msg RemoteConnectDoneMsg) (Model, tea.Cmd) {
	// Connection attempt finished: clear any "connecting" states for add-remote or remote auth.
	m.AddRemoteConnecting = false
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.RemoteAuthConnecting = false

	// When Remote Auth overlay is active, close it on successful connection.
	if m.RemoteAuthStep != "" {
		if msg.Success {
			m.OverlayActive = false
			m.OverlayTitle = ""
			m.OverlayContent = ""
			m.RemoteAuthStep = ""
			m.RemoteAuthTarget = ""
			m.RemoteAuthError = ""
			m.RemoteAuthUsername = ""
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
			m.Input.Focus()
		}
		return m, nil
	}

	// Fallback: add-remote overlay (opened via /remote on or /config add-remote).
	m.AddRemoteActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	if msg.Success {
		m.OverlayActive = false
		m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleRemoteAuthPromptMsg(msg RemoteAuthPromptMsg) (Model, tea.Cmd) {
	m.AddRemoteConnecting = false
	m.AddRemoteActive = false
	m.OverlayActive = true
	m.OverlayTitle = "Remote Auth"
	m.RemoteAuthTarget = msg.Target
	m.RemoteAuthError = msg.Err
	m.ChoiceIndex = 0
	// When UseConfiguredIdentity is true, show a non-interactive "connecting with configured key" state.
	if msg.UseConfiguredIdentity {
		m.RemoteAuthStep = "auto_identity"
		m.RemoteAuthConnecting = true
		return m, nil
	}
	// Default: interactive auth flow starting from username.
	m.RemoteAuthConnecting = false
	m.RemoteAuthStep = "username" // first step: username only; Enter then shows "choose" (1/2) so username can contain 1 or 2
	m.RemoteAuthUsernameInput = textinput.New()
	m.RemoteAuthUsernameInput.Placeholder = "root"
	if i := strings.Index(msg.Target, "@"); i > 0 && i < len(msg.Target)-1 {
		m.RemoteAuthUsernameInput.SetValue(msg.Target[:i])
	} else {
		m.RemoteAuthUsernameInput.SetValue("root")
	}
	m.RemoteAuthUsernameInput.Focus()
	return m, nil
}
