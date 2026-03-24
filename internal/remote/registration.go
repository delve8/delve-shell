package remote

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func registerSlashExactHandlers() {
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
}

func registerSlashPrefixHandlers() {
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
}

func registerProviders() {
	ui.RegisterSlashOptionsProvider(remoteSlashOptionsProvider)

	ui.RegisterOverlayKeyProvider(func(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
		return handleRemoteOverlayKey(m, key, msg)
	})

	ui.RegisterOverlayContentProvider(func(m ui.Model) (string, bool) {
		return buildRemoteOverlayContent(m)
	})

	ui.RegisterMessageProvider(remoteMessageProvider)
}
