package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

// registerTestSlashExactMirrors mirrors exact handlers registered by non-ui packages
// so internal/ui tests can run without importing those packages.
func registerTestSlashExactMirrors() {
	appendConfigHint := func(m Model) (Model, tea.Cmd) {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
		m = m.RefreshViewport()
		return m, nil
	}
	registerSlashExact("/config show", SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	registerSlashExact("/config", SlashExactDispatchEntry{
		Handle:     appendConfigHint,
		ClearInput: false,
	})
	registerSlashExact("/config update auto-run list", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.applyConfigAllowlistUpdate(), nil
		},
		ClearInput: true,
	})
	reloadConfig := func(m Model) (Model, tea.Cmd) {
		if m.Ports.ConfigUpdatedChan != nil {
			select {
			case m.Ports.ConfigUpdatedChan <- struct{}{}:
			default:
			}
		}
		return m, nil
	}
	registerSlashExact("/config reload", SlashExactDispatchEntry{
		Handle:     reloadConfig,
		ClearInput: true,
	})
	registerSlashExact("/reload", SlashExactDispatchEntry{
		Handle:     reloadConfig,
		ClearInput: true,
	})
	registerSlashExact("/cancel", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.Interaction.WaitingForAI && m.Ports.CancelRequestChan != nil {
				select {
				case m.Ports.CancelRequestChan <- struct{}{}:
				default:
				}
				m.Interaction.WaitingForAI = false
				return m, nil
			}
			lang := m.getLang()
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyNoRequestInProgress))))
			m = m.RefreshViewport()
			return m, nil
		},
		ClearInput: false,
	})
	registerSlashExact("/q", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) { return m, tea.Quit },
		ClearInput: false,
	})
	registerSlashExact("/sh", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
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
