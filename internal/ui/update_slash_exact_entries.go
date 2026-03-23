package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func init() {
	// Overlay-opening exact commands.
	registerSlashExact("/config llm", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openConfigLLMOverlay(), nil
		},
		ClearInput: true,
	})
	registerSlashExact("/config show", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		ClearInput: false,
	})
	registerSlashExact("/config", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		ClearInput: false,
	})
	// Connection toggles.
	// Agent cancel.
	registerSlashExact("/cancel", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.WaitingForAI && m.CancelRequestChan != nil {
				select {
				case m.CancelRequestChan <- struct{}{}:
				default:
				}
				m.WaitingForAI = false
				return m, nil
			}

			// Keep behavior consistent with the old direct handling branch.
			lang := m.getLang()
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyNoRequestInProgress))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		ClearInput: false,
	})

	// Config updates/reloads.
	registerSlashExact("/config update auto-run list", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.applyConfigAllowlistUpdate(), nil
		},
		ClearInput: true,
	})
	registerSlashExact("/config reload", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		ClearInput: true,
	})
	registerSlashExact("/reload", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		ClearInput: true,
	})

	// App lifecycle.
	registerSlashExact("/q", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m, tea.Quit
		},
		ClearInput: false,
	})
	registerSlashExact("/sh", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			if m.ShellRequestedChan != nil {
				msgs := make([]string, len(m.Messages))
				copy(msgs, m.Messages)
				select {
				case m.ShellRequestedChan <- msgs:
				default:
				}
			}
			return m, tea.Quit
		},
		ClearInput: false,
	})
}
