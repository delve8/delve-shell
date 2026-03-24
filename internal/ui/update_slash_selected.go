package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
