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
			m.RunCompletion.RemoteLabel = t.Label
			m.RunCompletion.RemoteCommands = nil
		} else {
			// Switching back to local: drop any remote /run completion cache.
			m.RunCompletion.RemoteLabel = ""
			m.RunCompletion.RemoteCommands = nil
		}
		m = m.RefreshViewport()
		return m, nil, true
	case ui.RunCompletionCacheMsg:
		// Remote cache update (sent by CLI on successful /remote on).
		// Ignore stale results from previous remotes.
		if t.RemoteLabel == "" || t.RemoteLabel != m.RemoteLabel {
			return m, nil, true
		}
		m.RunCompletion.RemoteLabel = t.RemoteLabel
		m.RunCompletion.RemoteCommands = t.Commands
		return m, nil, true
	case ui.RemoteConnectDoneMsg:
		m.AddRemote.Connecting = false
		m.AddRemote.Error = ""
		m.AddRemote.OfferOverwrite = false
		m.RemoteAuth.Connecting = false

		// When Remote Auth overlay is active, close it on successful connection.
		if m.RemoteAuth.Step != "" {
			if t.Success {
				m.OverlayActive = false
				m.OverlayTitle = ""
				m.OverlayContent = ""
				m.RemoteAuth.Step = ""
				m.RemoteAuth.Target = ""
				m.RemoteAuth.Error = ""
				m.RemoteAuth.Username = ""
				m.PathCompletion.Candidates = nil
				m.PathCompletion.Index = -1
				m.Input.Focus()
			}
			return m, nil, true
		}

		// Fallback: add-remote overlay.
		m.AddRemote.Active = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		if t.Success {
			m.OverlayActive = false
			m.Input.Focus()
		}
		return m, nil, true
	case ui.RemoteAuthPromptMsg:
		m.AddRemote.Connecting = false
		m.AddRemote.Active = false
		m.OverlayActive = true
		m.OverlayTitle = "Remote Auth"
		m.RemoteAuth.Target = t.Target
		m.RemoteAuth.Error = t.Err
		m.ChoiceIndex = 0
		if t.UseConfiguredIdentity {
			m.RemoteAuth.Step = "auto_identity"
			m.RemoteAuth.Connecting = true
			return m, nil, true
		}
		m.RemoteAuth.Connecting = false
		m.RemoteAuth.Step = "username"
		m.RemoteAuth.UsernameInput = textinput.New()
		m.RemoteAuth.UsernameInput.Placeholder = "root"
		if i := strings.Index(t.Target, "@"); i > 0 && i < len(t.Target)-1 {
			m.RemoteAuth.UsernameInput.SetValue(t.Target[:i])
		} else {
			m.RemoteAuth.UsernameInput.SetValue("root")
		}
		m.RemoteAuth.UsernameInput.Focus()
		return m, nil, true
	default:
		return m, nil, false
	}
}
