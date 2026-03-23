package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func init() {
	// Overlay-opening exact commands.
	registerSlashExact("/config llm", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			return m.openConfigLLMOverlay(), nil
		},
		clearInput: true,
	})
	registerSlashExact("/config show", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		clearInput: false,
	})
	registerSlashExact("/config", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		clearInput: false,
	})
	registerSlashExact("/config add-skill", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			return m.openAddSkillOverlay("", "", ""), nil
		},
		clearInput: true,
	})

	// Connection toggles.
	registerSlashExact("/remote off", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			if m.RemoteOffChan != nil {
				select {
				case m.RemoteOffChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		clearInput: true,
	})

	// Agent cancel.
	registerSlashExact("/cancel", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
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
		clearInput: false,
	})

	// Config updates/reloads.
	registerSlashExact("/config update auto-run list", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			return m.applyConfigAllowlistUpdate(), nil
		},
		clearInput: true,
	})
	registerSlashExact("/config reload", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		clearInput: true,
	})
	registerSlashExact("/reload", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
			return m, nil
		},
		clearInput: true,
	})

	// App lifecycle.
	registerSlashExact("/q", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			return m, tea.Quit
		},
		clearInput: false,
	})
	registerSlashExact("/sh", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
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
		clearInput: false,
	})
	registerSlashExact("/new", slashDispatchEntry{
		handle: func(m Model) (Model, tea.Cmd) {
			if m.SubmitChan != nil {
				m.SubmitChan <- "/new"
			}
			// /new consumes input and refreshes content.
			m = m.clearSlashInput()
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		},
		clearInput: false,
	})
}
