package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry.Entries() {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			return e.Handle(m, rest)
		}
	}
	return m, nil, false
}

// NOTE: slash prefix handlers are registered in feature packages.
