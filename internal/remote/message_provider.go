package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func remoteMessageProvider(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	switch t := msg.(type) {
	case OpenAddRemoteOverlayMsg:
		return openAddRemoteOverlay(m, t.Save, t.Connect), nil, true
	case ApplyConfigAddRemoteMsg:
		return applyConfigAddRemote(m, t.Args), nil, true
	case ApplyConfigRemoveRemoteMsg:
		return applyConfigRemoveRemote(m, t.NameOrTarget), nil, true
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
	case ConnectDoneMsg:
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
	case AuthPromptMsg:
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
