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

	ui.RegisterSlashExact("/cancel", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Interaction.WaitingForAI && m.Ports.CancelRequestChan != nil {
				select {
				case m.Ports.CancelRequestChan <- struct{}{}:
				default:
				}
				m.Interaction.WaitingForAI = false
				return m, nil
			}
			m.Messages = append(m.Messages, delveMsg("en", i18n.T("en", i18n.KeyNoRequestInProgress)))
			return m.RefreshViewport(), nil
		},
		ClearInput: false,
	})

	ui.RegisterSlashExact("/q", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) { return m, tea.Quit },
		ClearInput: false,
	})
	ui.RegisterSlashExact("/sh", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Ports.ShellRequestedChan != nil {
				msgs := make([]string, len(m.Messages))
				copy(msgs, m.Messages)
				select {
				case m.Ports.ShellRequestedChan <- msgs:
				default:
				}
			}
			return m, tea.Quit
		},
		ClearInput: false,
	})
}
