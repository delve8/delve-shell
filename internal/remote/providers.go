package remote

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func registerProviders() {
	ui.RegisterSlashOptionsProvider(remoteSlashOptionsProvider)
	ui.RegisterStateEventProvider(remoteStateProvider)
	ui.RegisterTitleBarFragmentProvider(remoteTitleBarFragment)

	ui.RegisterOverlayFeature(ui.OverlayFeature{
		KeyID: OverlayFeatureKey,
		Open: func(m *ui.Model, req ui.OverlayOpenRequest) (*ui.Model, tea.Cmd, bool) {
			if req.Key != OverlayOpenKeyAddRemote {
				return m, nil, false
			}
			m.OpenOverlayFeature(OverlayFeatureKey, i18n.T(i18n.KeyAddRemoteTitle), "")
			state := getRemoteOverlayState()
			state.AddRemote.Active = true
			state.RemoteAuth = RemoteAuthOverlayState{}
			state.AddRemote.Error = ""
			state.AddRemote.ChoiceIndex = 0
			state.AddRemote.OfferOverwrite = false
			state.AddRemote.Save = req.Params["save"] == "true"
			pathcomplete.SetState(pathcomplete.State{Index: -1})
			state.AddRemote.FieldIndex = 0
			state.AddRemote.HostInput = textinput.New()
			state.AddRemote.HostInput.Placeholder = i18n.T(i18n.KeyAddRemoteHostPlaceholder)
			state.AddRemote.HostInput.Focus()
			state.AddRemote.UserInput = textinput.New()
			state.AddRemote.UserInput.Placeholder = i18n.T(i18n.KeyAddRemoteUserPlaceholder)
			if lastUsername, err := config.LoadLastUsername(); err == nil && lastUsername != "" {
				state.AddRemote.UserInput.SetValue(lastUsername)
				state.AddRemote.UserInput.CursorEnd()
			}
			state.AddRemote.NameInput = textinput.New()
			state.AddRemote.NameInput.Placeholder = i18n.T(i18n.KeyAddRemoteNamePlaceholder)
			state.AddRemote.KeyInput = textinput.New()
			state.AddRemote.KeyInput.Placeholder = i18n.T(i18n.KeyAddRemoteKeyPlaceholder)
			if lastIdentityFile, err := config.LoadLastIdentityFile(); err == nil && lastIdentityFile != "" {
				state.AddRemote.KeyInput.SetValue(lastIdentityFile)
				state.AddRemote.KeyInput.CursorEnd()
			}
			setRemoteOverlayState(state)
			return m, nil, true
		},
		Key: func(m *ui.Model, key string, msg tea.KeyMsg) (*ui.Model, tea.Cmd, bool) {
			return handleRemoteOverlayKey(m, key, msg)
		},
		// AuthPromptMsg / ConnectDoneMsg are handled in [remoteStateProvider] so they apply when
		// no overlay is open yet (e.g. direct `/access <host>`).
		Content: func(m *ui.Model) (string, bool) {
			return buildRemoteOverlayContent(m)
		},
		Close: func(m *ui.Model, activeKey string) {
			if activeKey != OverlayFeatureKey {
				return
			}
			resetRemoteOverlayState()
			pathcomplete.ResetState()
		},
	})
}

func remoteTitleBarFragment(m *ui.Model) (string, bool) {
	if m.Remote.Offline {
		return i18n.T(i18n.KeyRemoteTitleBarOffline), true
	}
	if !m.Remote.Active {
		return "", false
	}
	if lbl := m.Remote.Label; lbl != "" {
		return i18n.T(i18n.KeyRemoteTitleBarRemote) + " " + lbl, true
	}
	return i18n.T(i18n.KeyRemoteTitleBarRemote), true
}
