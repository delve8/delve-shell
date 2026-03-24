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
	for _, p := range slashSelectedProviders {
		if m2, cmd, handled := p(m, chosen); handled {
			return m2, cmd, true
		}
	}

	entries := []slashSelectedEntry{
		{
			exact: "/run <cmd>",
			handle: func(mm Model, _ string) (Model, tea.Cmd, bool) {
				mm.Input.SetValue("/run ")
				mm.Input.CursorEnd()
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
