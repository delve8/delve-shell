package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleMainInputUpdate(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	m.syncSlashSuggestIndex()
	return m, cmd
}

// syncSlashSuggestIndex keeps slash suggestion selection in a valid range
// when the user edits slash input.
func (m *Model) syncSlashSuggestIndex() {
	if !strings.HasPrefix(m.Input.Value(), "/") {
		return
	}
	inputVal := m.Input.Value()
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	// Session list (Path set): do not reset index on every keystroke so user can pick another session with Enter
	if len(opts) == 0 || opts[0].Path == "" {
		m.SlashSuggestIndex = 0
	}
	if len(vis) > 0 && m.SlashSuggestIndex >= len(vis) {
		m.SlashSuggestIndex = 0
	}
}
