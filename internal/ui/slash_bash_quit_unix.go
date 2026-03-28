//go:build !windows

package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostcmd"
)

// trySlashBashQuit handles /bash: snapshot transcript and quit the TUI to spawn a shell.
func trySlashBashQuit(m Model, text string) (Model, tea.Cmd, bool) {
	if text != "/bash" {
		return m, nil, false
	}
	m = m.appendUserSubmittedEcho(text)
	mode := hostcmd.SubshellModeLocalBash
	if m.Remote.Active {
		mode = hostcmd.SubshellModeRemoteSSH
	}
	_ = m.EmitShellSnapshotIntentWithMode(m.TranscriptLines(), mode)
	return m.clearSlashInput(), tea.Quit, true
}
