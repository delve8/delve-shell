package remote

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashExact("/config add-remote", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return openAddRemoteOverlay(m, true, false), nil
		},
		ClearInput: true,
	})

	ui.RegisterSlashExact("/remote on", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return openAddRemoteOverlay(m, false, true), nil
		},
		ClearInput: true,
	})

	ui.RegisterSlashExact("/remote off", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.RemoteOffChan != nil {
				select {
				case m.RemoteOffChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		ClearInput: true,
	})

	ui.RegisterSlashPrefix("/config add-remote ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config add-remote ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			return applyConfigAddRemote(m, strings.TrimSpace(rest)), nil, true
		},
	})

	ui.RegisterSlashPrefix("/config del-remote ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config del-remote ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			nameOrTarget := strings.TrimSpace(rest)
			if nameOrTarget == "" {
				return m, nil, true
			}
			return applyConfigRemoveRemote(m, nameOrTarget), nil, true
		},
	})

	ui.RegisterSlashPrefix("/remote on ", ui.SlashPrefixDispatchEntry{
		Prefix: "/remote on ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			target := strings.TrimSpace(rest)
			if target == "" {
				return m, nil, true
			}
			if m.RemoteOnChan != nil {
				select {
				case m.RemoteOnChan <- target:
				default:
				}
			}
			return m, nil, true
		},
	})

	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		_ string,
		_ []string,
		_ []string,
		_ bool,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)

		if normalizedLower == "remote" {
			return []ui.SlashOption{
				{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyDescRemoteOn), Path: ""},
				{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff), Path: ""},
			}, true
		}

		if strings.HasPrefix(normalizedLower, "remote on") {
			filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "remote on"))
			opts := getRemoteSlashOptions(filter, lang)
			offOpt := ui.SlashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff), Path: ""}
			return append([]ui.SlashOption{offOpt}, opts...), true
		}

		if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
			rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
			if rest == "del-remote" || strings.HasPrefix(rest, "del-remote ") {
				filter := strings.TrimSpace(strings.TrimPrefix(rest, "del-remote"))
				return getRemoveRemoteSlashOptions(lang, filter), true
			}
		}

		return nil, false
	})

	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		return handleRemoteOverlayKey(m, key, msg)
	})

	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildRemoteOverlayContent(m)
	})

	// Delegate remote async messages (connect done / auth prompt) to ui handlers.
	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
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
	})
}

func getRemoteSlashOptions(filter string, lang string) []ui.SlashOption {
	var opts []ui.SlashOption
	remotes, err := config.LoadRemotes()
	if err == nil && len(remotes) > 0 {
		filterLower := strings.ToLower(filter)
		for _, r := range remotes {
			line := r.Target + " " + r.Name
			if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
				continue
			}
			desc := r.Name
			opts = append(opts, ui.SlashOption{
				Cmd:  "/remote on " + config.HostFromTarget(r.Target),
				Desc: desc,
				Path: "",
			})
		}
	}

	manual := ui.SlashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyRemoteManualHint), Path: ""}
	return append(opts, manual)
}

func getRemoveRemoteSlashOptions(lang string, filter string) []ui.SlashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}

	filterLower := strings.ToLower(filter)
	var opts []ui.SlashOption
	for _, r := range remotes {
		line := r.Target + " " + r.Name
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}

		desc := r.Name
		opts = append(opts, ui.SlashOption{
			Cmd:  "/config del-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
			Path: "",
		})
	}

	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}
	return opts
}
