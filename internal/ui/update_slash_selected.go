package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) showConfigHint() Model {
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyConfigHint))))
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m
}

// handleSlashSelectedFallback handles suggestion-selected slash commands
// that are intentionally not routed through exact/prefix dispatcher.
func (m Model) handleSlashSelectedFallback(chosen string) (Model, tea.Cmd, bool) {
	if chosen == "/run <cmd>" {
		m.Input.SetValue("/run ")
		m.Input.CursorEnd()
		return m, nil, true
	}
	if strings.HasPrefix(chosen, "/skill ") {
		// Fill so user can type natural language after the skill name.
		m.Input.SetValue(chosen + " ")
		m.Input.CursorEnd()
		m.SlashSuggestIndex = 0
		return m, nil, true
	}
	if strings.HasPrefix(chosen, "/config add-remote ") {
		m.Input.SetValue("/config add-remote ")
		m.Input.CursorEnd()
		return m, nil, true
	}
	if strings.HasPrefix(chosen, "/config del-remote ") {
		nameOrTarget := strings.TrimSpace(strings.TrimPrefix(chosen, "/config del-remote "))
		if nameOrTarget != "" {
			return m.applyConfigRemoveRemote(nameOrTarget), nil, true
		}
		m.Input.SetValue("/config del-remote ")
		m.Input.CursorEnd()
		return m, nil, true
	}
	if chosen == "/config del-remote" {
		m.Input.SetValue("/config del-remote ")
		m.Input.CursorEnd()
		return m, nil, true
	}
	if chosen == "/config" || strings.HasPrefix(chosen, "/config ") {
		return m.showConfigHint(), nil, true
	}
	return m, nil, false
}
