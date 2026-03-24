package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostnotify"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	appendConfigHint := func(m ui.Model) (ui.Model, tea.Cmd) {
		m.Messages = append(m.Messages, delveMsg("en", i18n.T("en", i18n.KeyConfigHint)))
		return m.RefreshViewport(), nil
	}
	triggerConfigReload := func(m ui.Model) (ui.Model, tea.Cmd) {
		hostnotify.NotifyConfigUpdated()
		return m, nil
	}

	ui.RegisterSlashExact("/config show", ui.SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: true,
	})
	ui.RegisterSlashExact("/config update auto-run list", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return applyConfigAllowlistUpdate(m), nil
		},
		ClearInput: true,
	})
	ui.RegisterSlashExact("/config reload", ui.SlashExactDispatchEntry{
		Handle:     triggerConfigReload,
		ClearInput: true,
	})
	ui.RegisterSlashExact("/reload", ui.SlashExactDispatchEntry{
		Handle:     triggerConfigReload,
		ClearInput: true,
	})
}
