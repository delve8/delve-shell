package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func remoteMessageProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	switch t := msg.(type) {
	case ExecutionChangedMsg:
		m.Remote.Active = t.Active
		m.Remote.Label = t.Label
		// Remote execution state changed: clear any previously cached /run suggestions.
		clearCachedRunSuggestions()
		m = m.RefreshViewport()
		return m, nil, true
	case RunCompletionCacheMsg:
		// Remote cache update (sent by CLI on successful /remote on).
		// Ignore stale results from previous remotes.
		if t.RemoteLabel == "" || t.RemoteLabel != m.Remote.Label {
			return m, nil, true
		}
		setCachedRunSuggestions(t.Commands)
		return m, nil, true
	default:
		return m, nil, false
	}
}

func remoteOverlayEventProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	if m.Overlay.Key != "remote" {
		return m, nil, false
	}

	state := getRemoteOverlayState()
	switch t := msg.(type) {
	case ConnectDoneMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
		state.RemoteAuth.Connecting = false

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

		state.AddRemote.Active = false
		if t.Success {
			m = m.CloseOverlayVisual()
			m.Input.Focus()
		}
		setRemoteOverlayState(state)
		return m, nil, true
	case AuthPromptMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.Active = false
		m = m.OpenOverlayFeature("remote", "Remote Auth", "")
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
