package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostnotify"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func remoteMessageProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	switch t := msg.(type) {
	case ui.RemoteStatusMsg:
		hostnotify.SetRemoteExecution(t.Active, t.Label)
		if t.Active {
			// New remote active: clear any previous remote /run completion cache.
			m.RunCompletion.RemoteCommands = nil
		} else {
			// Switching back to local: drop any remote /run completion cache.
			m.RunCompletion.RemoteCommands = nil
		}
		m = m.RefreshViewport()
		return m, nil, true
	case ui.RunCompletionCacheMsg:
		// Remote cache update (sent by CLI on successful /remote on).
		// Ignore stale results from previous remotes.
		if t.RemoteLabel == "" || t.RemoteLabel != hostnotify.RemoteLabel() {
			return m, nil, true
		}
		m.RunCompletion.RemoteCommands = t.Commands
		return m, nil, true
	case ui.RemoteConnectDoneMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
		state.RemoteAuth.Connecting = false

		// When Remote Auth overlay is active, close it on successful connection.
		if state.RemoteAuth.Step != "" {
			if t.Success {
				m = m.CloseOverlayVisual()
				state.RemoteAuth.Step = ""
				state.RemoteAuth.Target = ""
				state.RemoteAuth.Error = ""
				state.RemoteAuth.Username = ""
				pathcomplete.SetState(pathcomplete.State{Index: -1})
				m.Input.Focus()
			}
			setRemoteOverlayState(state)
			return m, nil, true
		}

		// Fallback: add-remote overlay.
		state.AddRemote.Active = false
		m.Overlay.Title = ""
		m.Overlay.Content = ""
		if t.Success {
			m.Overlay.Active = false
			m.Input.Focus()
		}
		setRemoteOverlayState(state)
		return m, nil, true
	case ui.RemoteAuthPromptMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.Active = false
		m.Overlay.Active = true
		m.Overlay.Title = "Remote Auth"
		state.RemoteAuth.Target = t.Target
		state.RemoteAuth.Error = t.Err
		m.Interaction.ChoiceIndex = 0
		if t.UseConfiguredIdentity {
			state.RemoteAuth.Step = "auto_identity"
			state.RemoteAuth.Connecting = true
			setRemoteOverlayState(state)
			return m, nil, true
		}
		state.RemoteAuth.Connecting = false
		state.RemoteAuth.Step = "username"
		state.RemoteAuth.UsernameInput = textinput.New()
		state.RemoteAuth.UsernameInput.Placeholder = "root"
		if i := strings.Index(t.Target, "@"); i > 0 && i < len(t.Target)-1 {
			state.RemoteAuth.UsernameInput.SetValue(t.Target[:i])
		} else {
			state.RemoteAuth.UsernameInput.SetValue("root")
		}
		state.RemoteAuth.UsernameInput.Focus()
		setRemoteOverlayState(state)
		return m, nil, true
	default:
		return m, nil, false
	}
}
