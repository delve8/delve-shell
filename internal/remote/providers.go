package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/host/cmd"
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
			connectOnly := req.Params["connect"] == "true"
			title := i18n.T(i18n.KeyAddRemoteTitle)
			if connectOnly {
				title = i18n.T(i18n.KeyConnectRemoteTitle)
			}
			m.OpenOverlayFeature(OverlayFeatureKey, title, "")
			state := getRemoteOverlayState()
			state.AddRemote = AddRemoteOverlayState{}
			state.ConnectRemote = RemoteConnectOverlayState{}
			state.RemoteAuth = RemoteAuthOverlayState{}
			if connectOnly {
				target := strings.TrimSpace(req.Params["target"])
				state.ConnectRemote = RemoteConnectOverlayState{
					Active:     true,
					Target:     target,
					Connecting: true,
				}
				setRemoteOverlayState(state)
				if m.CommandSender == nil || !m.CommandSender.Send(hostcmd.AccessRemote{Target: target}) {
					state.ConnectRemote.Connecting = false
					state.ConnectRemote.Error = "failed to start remote connection"
					setRemoteOverlayState(state)
				}
				return m, nil, true
			}
			state.AddRemote.Active = true
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
			state.AddRemote.Socks5Input = textinput.New()
			state.AddRemote.Socks5Input.Placeholder = i18n.T(i18n.KeyAddRemoteSocks5Placeholder)
			if lastSocks5Addr, err := config.LoadLastSocks5Addr(); err == nil && lastSocks5Addr != "" {
				state.AddRemote.Socks5Input.SetValue(lastSocks5Addr)
				state.AddRemote.Socks5Input.CursorEnd()
			}
			prefillAddRemoteFromParams(&state.AddRemote, req.Params)
			applyAddRemoteFieldFocus(&state.AddRemote)
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

func resolveConnectTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if remotes, err := config.LoadRemotes(); err == nil {
		for _, r := range remotes {
			matched := r.Target == target || config.HostFromTarget(r.Target) == target
			if !matched || strings.TrimSpace(r.Target) == "" {
				continue
			}
			return strings.TrimSpace(r.Target)
		}
	}
	if sshHost, ok, err := config.ResolveSSHConfigHost(target); err == nil && ok {
		return strings.TrimSpace(sshHost.Target)
	}
	if remotes, err := config.LoadRemotes(); err == nil {
		for _, r := range remotes {
			if strings.TrimSpace(r.Name) == target && strings.TrimSpace(r.Target) != "" {
				return strings.TrimSpace(r.Target)
			}
		}
	}
	return target
}

func prefillAddRemoteFromParams(state *AddRemoteOverlayState, params map[string]string) {
	if state == nil {
		return
	}
	target := strings.TrimSpace(params["target"])
	if target == "" {
		return
	}
	resolvedTarget := resolveConnectTarget(target)
	if remotes, err := config.LoadRemotes(); err == nil {
		for _, r := range remotes {
			matched := r.Target == target || r.Name == target || config.HostFromTarget(r.Target) == target
			if !matched || strings.TrimSpace(r.Target) == "" {
				continue
			}
			resolvedTarget = strings.TrimSpace(r.Target)
			if strings.TrimSpace(r.IdentityFile) != "" {
				state.KeyInput.SetValue(strings.TrimSpace(r.IdentityFile))
				state.KeyInput.CursorEnd()
			}
			if strings.TrimSpace(r.Socks5Addr) != "" {
				state.Socks5Input.SetValue(strings.TrimSpace(r.Socks5Addr))
				state.Socks5Input.CursorEnd()
			}
			if strings.TrimSpace(r.Name) != "" {
				state.NameInput.SetValue(strings.TrimSpace(r.Name))
				state.NameInput.CursorEnd()
			}
			break
		}
	}
	if state.HostInput.Value() == "" && state.UserInput.Value() == "" {
		if sshHost, ok, err := config.ResolveSSHConfigHost(target); err == nil && ok {
			if strings.TrimSpace(sshHost.IdentityFile) != "" {
				state.KeyInput.SetValue(strings.TrimSpace(sshHost.IdentityFile))
				state.KeyInput.CursorEnd()
			}
			if strings.TrimSpace(sshHost.Alias) != "" {
				state.NameInput.SetValue(strings.TrimSpace(sshHost.Alias))
				state.NameInput.CursorEnd()
			}
			resolvedTarget = strings.TrimSpace(sshHost.Target)
		}
	}
	if i := strings.Index(resolvedTarget, "@"); i > 0 && i < len(resolvedTarget)-1 {
		state.UserInput.SetValue(strings.TrimSpace(resolvedTarget[:i]))
		state.UserInput.CursorEnd()
		host := strings.TrimSpace(resolvedTarget[i+1:])
		state.HostInput.SetValue(host)
		state.HostInput.CursorEnd()
		return
	}
	state.HostInput.SetValue(config.HostFromTarget(resolvedTarget))
	state.HostInput.CursorEnd()
	state.FieldIndex = 1
}

func resolveRemoteIdentityPrefill(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if remotes, err := config.LoadRemotes(); err == nil {
		for _, r := range remotes {
			matched := r.Target == target || r.Name == target || config.HostFromTarget(r.Target) == config.HostFromTarget(target)
			if !matched {
				continue
			}
			if identity := strings.TrimSpace(r.IdentityFile); identity != "" {
				return identity
			}
			break
		}
	}
	if sshHosts, err := config.LoadSSHConfigHosts(); err == nil {
		targetHost := config.HostFromTarget(target)
		for _, h := range sshHosts {
			matched := strings.EqualFold(h.Alias, target) ||
				strings.EqualFold(h.Target, target) ||
				strings.EqualFold(config.HostFromTarget(h.Target), targetHost)
			if !matched {
				continue
			}
			if identity := strings.TrimSpace(h.IdentityFile); identity != "" {
				return identity
			}
			break
		}
	}
	if lastIdentityFile, err := config.LoadLastIdentityFile(); err == nil {
		return strings.TrimSpace(lastIdentityFile)
	}
	return ""
}

func remoteTitleBarFragment(m *ui.Model) (string, bool) {
	if m.Remote.Offline {
		return i18n.T(i18n.KeyRemoteTitleBarOffline), true
	}
	if !m.Remote.Active {
		return "", false
	}
	if issue := strings.TrimSpace(m.Remote.Issue); issue != "" {
		if lbl := m.Remote.Label; lbl != "" {
			return i18n.T(i18n.KeyRemoteTitleBarRemote) + " " + lbl + " (" + issue + ")", true
		}
		return i18n.T(i18n.KeyRemoteTitleBarRemote) + " (" + issue + ")", true
	}
	if lbl := m.Remote.Label; lbl != "" {
		return i18n.T(i18n.KeyRemoteTitleBarRemote) + " " + lbl, true
	}
	return i18n.T(i18n.KeyRemoteTitleBarRemote), true
}
