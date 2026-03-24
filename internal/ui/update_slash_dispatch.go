package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) clearSlashInput() Model {
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.SlashSuggestIndex = 0
	return m
}

// dispatchSlashExact routes exact slash commands through a single table-driven path.
// clearInput controls whether the slash input is consumed after execution.
func (m Model) dispatchSlashExact(cmd string) (Model, tea.Cmd, bool) {
	entry, ok := slashExactDispatchRegistry[cmd]
	if !ok {
		return m, nil, false
	}
	m, outCmd := entry.Handle(m)
	if entry.ClearInput {
		m = m.clearSlashInput()
	}
	return m, outCmd, true
}
