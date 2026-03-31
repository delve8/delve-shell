package remote

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func registerProviders() {
	ui.RegisterSlashOptionsProvider(remoteSlashOptionsProvider)
	ui.RegisterStateEventProvider(remoteStateProvider)

	ui.RegisterTitleBarFragmentProvider(func(m ui.Model) (string, bool) {
		if m.Remote.Offline {
			return "Offline", true
		}
		if !m.Remote.Active {
			return "", false
		}
		if lbl := m.Remote.Label; lbl != "" {
			return "Remote " + lbl, true
		}
		return "Remote", true
	})

	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: "remote",
		Open: func(m ui.Model, req ui.OverlayOpenRequest) (ui.Model, tea.Cmd, bool) {
			if req.Key != "remote_add" {
				return m, nil, false
			}
			m = m.OpenOverlayFeature("remote", i18n.T(i18n.KeyAddRemoteTitle), "")
			state := getRemoteOverlayState()
			state.AddRemote.Active = true
			state.RemoteAuth = RemoteAuthOverlayState{}
			state.AddRemote.Error = ""
			state.AddRemote.OfferOverwrite = false
			state.AddRemote.Save = req.Params["save"] == "true"
			pathcomplete.SetState(pathcomplete.State{Index: -1})
			state.AddRemote.FieldIndex = 0
			state.AddRemote.HostInput = textinput.New()
			state.AddRemote.HostInput.Placeholder = "host or host:22"
			state.AddRemote.HostInput.Focus()
			state.AddRemote.UserInput = textinput.New()
			state.AddRemote.UserInput.Placeholder = "e.g. root"
			state.AddRemote.UserInput.SetValue("root")
			state.AddRemote.NameInput = textinput.New()
			state.AddRemote.NameInput.Placeholder = "name (optional)"
			state.AddRemote.KeyInput = textinput.New()
			state.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
			setRemoteOverlayState(state)
			return m, nil, true
		},
		Key: func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
			return handleRemoteOverlayKey(m, key, msg)
		},
		// AuthPromptMsg / ConnectDoneMsg are handled in [remoteStateProvider] so they apply when
		// no overlay is open yet (e.g. direct `/access <host>`).
		Content: func(m ui.Model) (string, bool) {
			return buildRemoteOverlayContent(m)
		},
		Close: func(m ui.Model, activeKey string) ui.Model {
			if activeKey != "remote" {
				return m
			}
			resetRemoteOverlayState()
			pathcomplete.ResetState()
			return m
		},
	})
}
