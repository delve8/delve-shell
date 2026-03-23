package remote

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashExact("/config add-remote", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return m.OpenAddRemoteOverlay(true, false), nil
		},
		ClearInput: true,
	})

	ui.RegisterSlashExact("/remote on", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return m.OpenAddRemoteOverlay(false, true), nil
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
			return m.ApplyConfigAddRemoteArgs(strings.TrimSpace(rest)), nil, true
		},
	})

	ui.RegisterSlashPrefix("/config del-remote ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config del-remote ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			nameOrTarget := strings.TrimSpace(rest)
			if nameOrTarget == "" {
				return m, nil, true
			}
			return m.ApplyConfigRemoveRemote(nameOrTarget), nil, true
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

	// Delegate remote async messages (connect done / auth prompt) to ui handlers.
	ui.RegisterMessageProvider(func(m ui.Model, msg tea.Msg) (ui.Model, tea.Cmd, bool) {
		switch t := msg.(type) {
		case ui.RemoteConnectDoneMsg:
			m2, cmd := m.HandleRemoteConnectDoneMsg(t)
			return m2, cmd, true
		case ui.RemoteAuthPromptMsg:
			m2, cmd := m.HandleRemoteAuthPromptMsg(t)
			return m2, cmd, true
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
