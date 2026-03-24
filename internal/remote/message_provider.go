package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func remoteMessageProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	switch t := msg.(type) {
	case ui.RemoteStatusMsg:
		m.RemoteActive = t.Active
		m.RemoteLabel = t.Label
		if t.Active {
			// New remote active: clear any previous remote /run completion cache.
			m.RemoteRunLabel = t.Label
			m.RemoteRunCommands = nil
		} else {
			// Switching back to local: drop any remote /run completion cache.
			m.RemoteRunLabel = ""
			m.RemoteRunCommands = nil
		}
		m = m.RefreshViewport()
		return m, nil, true
	case ui.RunCompletionCacheMsg:
		// Remote cache update (sent by CLI on successful /remote on).
		// Ignore stale results from previous remotes.
		if t.RemoteLabel == "" || t.RemoteLabel != m.RemoteLabel {
			return m, nil, true
		}
		m.RemoteRunLabel = t.RemoteLabel
		m.RemoteRunCommands = t.Commands
		return m, nil, true
	case ui.RemoteConnectDoneMsg:
		m.AddRemoteConnecting = false
		m.AddRemoteError = ""
		m.AddRemoteOfferOverwrite = false
		m.RemoteAuthConnecting = false

		// When Remote Auth overlay is active, close it on successful connection.
		if m.RemoteAuthStep != "" {
			if t.Success {
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
			return m, nil, true
		}

		// Fallback: add-remote overlay.
		m.AddRemoteActive = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		if t.Success {
			m.OverlayActive = false
			m.Input.Focus()
		}
		return m, nil, true
	case ui.RemoteAuthPromptMsg:
		m.AddRemoteConnecting = false
		m.AddRemoteActive = false
		m.OverlayActive = true
		m.OverlayTitle = "Remote Auth"
		m.RemoteAuthTarget = t.Target
		m.RemoteAuthError = t.Err
		m.ChoiceIndex = 0
		if t.UseConfiguredIdentity {
			m.RemoteAuthStep = "auto_identity"
			m.RemoteAuthConnecting = true
			return m, nil, true
		}
		m.RemoteAuthConnecting = false
		m.RemoteAuthStep = "username"
		m.RemoteAuthUsernameInput = textinput.New()
		m.RemoteAuthUsernameInput.Placeholder = "root"
		if i := strings.Index(t.Target, "@"); i > 0 && i < len(t.Target)-1 {
			m.RemoteAuthUsernameInput.SetValue(t.Target[:i])
		} else {
			m.RemoteAuthUsernameInput.SetValue("root")
		}
		m.RemoteAuthUsernameInput.Focus()
		return m, nil, true
	default:
		return m, nil, false
	}
}
