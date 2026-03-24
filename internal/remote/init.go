package remote

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"

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
	ui.RegisterSlashExact("/config del-remote", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			m.Input.SetValue("/config del-remote ")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			return m, nil
		},
		ClearInput: false,
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

	ui.RegisterMessageProvider(remoteMessageProvider)
}
