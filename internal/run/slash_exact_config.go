package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	appendConfigHint := func(m ui.Model) (ui.Model, tea.Cmd) {
		m.Messages = append(m.Messages, delveMsg("en", i18n.T("en", i18n.KeyConfigHint)))
		return m.RefreshViewport(), nil
	}
	triggerConfigReload := func(m ui.Model) (ui.Model, tea.Cmd) {
		if m.Ports.ConfigUpdatedChan != nil {
			select {
			case m.Ports.ConfigUpdatedChan <- struct{}{}:
			default:
			}
		}
		return m, nil
	}

	ui.RegisterSlashExact("/config show", ui.SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	ui.RegisterSlashExact("/config", ui.SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	ui.RegisterSlashExact("/config update auto-run list", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return m.ApplyConfigAllowlistUpdate(), nil
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
