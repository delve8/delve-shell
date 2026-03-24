package ui

import tea "github.com/charmbracelet/bubbletea"

// handleSlashSelectedFallback handles suggestion-selected slash commands
// that are intentionally not routed through exact/prefix dispatcher.
func (m Model) handleSlashSelectedFallback(chosen string) (Model, tea.Cmd, bool) {
	for _, p := range slashSelectedProviderChain.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			return m2, cmd, true
		}
	}
	return m, nil, false
}
