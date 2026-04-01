//go:build windows

package ui

import tea "github.com/charmbracelet/bubbletea"

// trySlashBashQuit is a no-op on Windows; /bash is not offered in the slash list.
func trySlashBashQuit(m *Model, _ string) (*Model, tea.Cmd, bool) {
	return m, nil, false
}
