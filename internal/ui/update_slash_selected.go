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

type slashSelectedEntry struct {
	exact  string
	prefix string
	handle func(Model, string) (Model, tea.Cmd, bool) // chosen (full) passed
}

// handleSlashSelectedFallback handles suggestion-selected slash commands
// that are intentionally not routed through exact/prefix dispatcher.
func (m Model) handleSlashSelectedFallback(chosen string) (Model, tea.Cmd, bool) {
	entries := []slashSelectedEntry{
		{
			exact: "/run <cmd>",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				mm.Input.SetValue("/run ")
				mm.Input.CursorEnd()
				return mm, nil, true
			},
		},
		{
			prefix: "/skill ",
			handle: func(mm Model, chosen string) (Model, tea.Cmd, bool) {
				// Fill so user can type natural language after the skill name.
				mm.Input.SetValue(chosen + " ")
				mm.Input.CursorEnd()
				mm.SlashSuggestIndex = 0
				return mm, nil, true
			},
		},
		{
			prefix: "/config add-remote ",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				mm.Input.SetValue("/config add-remote ")
				mm.Input.CursorEnd()
				return mm, nil, true
			},
		},
		{
			prefix: "/config del-remote ",
			handle: func(mm Model, chosen string) (Model, tea.Cmd, bool) {
				nameOrTarget := strings.TrimSpace(strings.TrimPrefix(chosen, "/config del-remote "))
				if nameOrTarget != "" {
					return mm.applyConfigRemoveRemote(nameOrTarget), nil, true
				}
				mm.Input.SetValue("/config del-remote ")
				mm.Input.CursorEnd()
				return mm, nil, true
			},
		},
		{
			exact: "/config del-remote",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				mm.Input.SetValue("/config del-remote ")
				mm.Input.CursorEnd()
				return mm, nil, true
			},
		},
		{
			exact: "/config",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				return mm.showConfigHint(), nil, true
			},
		},
		{
			prefix: "/config ",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				return mm.showConfigHint(), nil, true
			},
		},
	}

	for _, e := range entries {
		if e.exact != "" && chosen == e.exact {
			return e.handle(m, chosen)
		}
		if e.prefix != "" && strings.HasPrefix(chosen, e.prefix) {
			return e.handle(m, chosen)
		}
	}
	return m, nil, false
}
