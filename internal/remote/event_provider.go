package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func remoteStateProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	switch t := msg.(type) {
	case ExecutionChangedMsg:
		m.Remote.Active = t.Active
		m.Remote.Label = t.Label
		m.Remote.Offline = t.Offline
		clearCachedRunSuggestions()
		return m, nil, true
	case RunCompletionCacheMsg:
		if t.RemoteLabel == "" || t.RemoteLabel != m.Remote.Label {
			return m, nil, true
		}
		setCachedRunSuggestions(t.Commands)
		return m, nil, true
	case AuthPromptMsg, ConnectDoneMsg:
		// Must run here, not only via overlay feature Event: direct `/access …` has no overlay
		// open yet; those messages were previously dropped and produced a silent failure.
		return remoteConnectUIHandler(m, msg)
	default:
		return m, nil, false
	}
}

func remoteConnectUIHandler(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
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
				state.RemoteAuth.HostKeyHost = ""
				state.RemoteAuth.HostKeyFP = ""
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
		state.RemoteAuth.HostKeyHost = t.HostKeyHost
		state.RemoteAuth.HostKeyFP = t.HostKeyFingerprint
		m.Interaction.ChoiceIndex = 0
		if t.HostKeyVerify {
			state.RemoteAuth.Step = "hostkey"
			state.RemoteAuth.Connecting = false
			setRemoteOverlayState(state)
			return m, nil, true
		}
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
