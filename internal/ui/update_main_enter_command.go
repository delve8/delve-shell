package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) handleMainEnterCommand(text string, slashSelectedPath string, slashSelectedIndex int) (Model, tea.Cmd) {
	if strings.HasPrefix(text, "/") {
		if m2, cmd, handled := m.dispatchSlashExact(text); handled {
			return m2, cmd
		}
	}
	if m2, cmd, handled := m.dispatchSlashPrefix(text); handled {
		return m2, cmd
	}

	if strings.HasPrefix(text, "/") {
		// Use path captured before SlashSuggestIndex was reset; otherwise we would always send opts[0].
		if slashSelectedPath != "" {
			if m.Ports.SessionSwitchChan != nil {
				select {
				case m.Ports.SessionSwitchChan <- slashSelectedPath:
				default:
				}
			}
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			return m, nil
		}

		opts := getSlashOptionsForInput(text, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
		vis := visibleSlashOptions(text, opts)

		var selectedOpt slashOption
		if slashSelectedIndex >= 0 && slashSelectedIndex < len(vis) {
			selectedOpt = opts[vis[slashSelectedIndex]]
		}

		// Sessions list empty: show message only when the single option is the session-none placeholder (not for del-skill etc).
		sessionNoneMsg := i18n.T(m.getLang(), i18n.KeySessionNone)
		if selectedOpt.Path == "" && len(vis) == 1 && selectedOpt.Cmd == sessionNoneMsg {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(sessionNoneMsg)))
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Input.SetValue("")
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0
			return m, nil
		}

		chosen := selectedOpt.Cmd

		// input must match chosen command; skip when only "/". "Fill only" already returned above.
		if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && (chosen == text || strings.HasPrefix(chosen, text)) {
			// user input matches chosen (full input then Enter) => execute
			// Slash suggestion for "/skill <name>" is "fill-only":
			// skip execution dispatch when there is no natural language yet.
			if strings.HasPrefix(chosen, "/skill ") {
				rest := strings.TrimSpace(strings.TrimPrefix(chosen, "/skill "))
				fields := strings.Fields(rest)
				if len(fields) == 1 {
					if m2, cmd, handled := m.handleSlashSelectedFallback(chosen); handled {
						return m2, cmd
					}
				}
			}

			if m2, cmd, handled := m.dispatchSlashExact(chosen); handled {
				return m2, cmd
			}
			if m2, cmd, handled := m.dispatchSlashPrefix(chosen); handled {
				return m2, cmd
			}
			if m2, cmd, handled := m.handleSlashSelectedFallback(chosen); handled {
				return m2, cmd
			}
		}

		m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyUnknownCmd))))
		m.Viewport.SetContent(m.buildContent())
		m.Viewport.GotoBottom()
		return m, nil
	}

	if m.Ports.SubmitChan != nil {
		m.Ports.SubmitChan <- text
		m.WaitingForAI = true
	}
	return m, nil
}
