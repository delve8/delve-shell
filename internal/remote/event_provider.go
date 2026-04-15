package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func remoteStateProvider(m *ui.Model, msg tea.Msg) (*ui.Model, tea.Cmd, bool) {
	switch t := msg.(type) {
	case ExecutionChangedMsg:
		m.Remote.Active = t.Active
		m.Remote.Label = t.Label
		m.Remote.Offline = t.Offline
		m.Remote.Issue = strings.TrimSpace(t.Issue)
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

func remoteConnectUIHandler(m *ui.Model, msg tea.Msg) (*ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	switch t := msg.(type) {
	case ConnectDoneMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.OfferOverwrite = false
		state.ConnectRemote.Connecting = false
		state.ConnectRemote.Error = t.Err
		state.RemoteAuth.Connecting = false

		if state.RemoteAuth.Step != "" {
			state.RemoteAuth.Error = t.Err
			if t.Success {
				cmd := m.CloseOverlayAndRefocusInput()
				state.RemoteAuth.Step = ""
				state.RemoteAuth.Target = ""
				state.RemoteAuth.Error = ""
				state.RemoteAuth.HostKeyHost = ""
				state.RemoteAuth.HostKeyFP = ""
				state.RemoteAuth.Username = ""
				pathcomplete.SetState(pathcomplete.State{Index: -1})
				setRemoteOverlayState(state)
				return m, cmd, true
			}
			setRemoteOverlayState(state)
			return m, nil, true
		}

		if state.ConnectRemote.Active {
			if t.Success {
				state.ConnectRemote = RemoteConnectOverlayState{}
				setRemoteOverlayState(state)
				return m, m.CloseOverlayAndRefocusInput(), true
			}
			setRemoteOverlayState(state)
			return m, nil, true
		}

		state.AddRemote.Error = t.Err
		if t.Success {
			state.AddRemote.Active = false
			setRemoteOverlayState(state)
			return m, m.CloseOverlayAndRefocusInput(), true
		} else if state.AddRemote.Active {
			applyAddRemoteFieldFocus(&state.AddRemote)
		}
		setRemoteOverlayState(state)
		return m, nil, true

	case AuthPromptMsg:
		state.AddRemote.Connecting = false
		state.AddRemote.Active = false
		state.ConnectRemote.Connecting = false
		state.ConnectRemote.Active = false
		m.OpenOverlayFeature(OverlayFeatureKey, i18n.T(i18n.KeyRemoteAuthTitle), "")
		state.RemoteAuth.Target = t.Target
		state.RemoteAuth.Error = t.Err
		state.RemoteAuth.ChoiceIndex = 0
		state.RemoteAuth.HostKeyHost = t.HostKeyHost
		state.RemoteAuth.HostKeyFP = t.HostKeyFingerprint
		m.Interaction.ChoiceIndex = 0
		if t.HostKeyVerify {
			state.RemoteAuth.Step = AuthStepHostKey
			state.RemoteAuth.Connecting = false
			setRemoteOverlayState(state)
			return m, nil, true
		}
		if t.UseConfiguredIdentity {
			state.RemoteAuth.Step = AuthStepAutoIdentity
			state.RemoteAuth.Connecting = true
			setRemoteOverlayState(state)
			return m, nil, true
		}
		state.RemoteAuth.Connecting = false
		state.RemoteAuth.Step = AuthStepUsername
		state.RemoteAuth.UsernameInput = textinput.New()
		state.RemoteAuth.UsernameInput.Placeholder = i18n.T(i18n.KeyAddRemoteUserPlaceholder)
		if i := strings.Index(t.Target, "@"); i > 0 && i < len(t.Target)-1 {
			state.RemoteAuth.UsernameInput.SetValue(t.Target[:i])
		}
		state.RemoteAuth.UsernameInput.Focus()
		setRemoteOverlayState(state)
		return m, nil, true

	default:
		return m, nil, false
	}
}
