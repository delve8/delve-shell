//go:build !windows

package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
)

// trySlashBashQuit handles /bash: snapshot transcript and quit the TUI to spawn a shell.
func trySlashBashQuit(m *Model, text string) (*Model, tea.Cmd, bool) {
	if text != "/bash" {
		return m, nil, false
	}
	if m.offlineExecutionMode() {
		m.AppendTranscriptLines(errStyle.Render(i18n.T(i18n.KeyOfflineExecBashDisabled)))
		m.clearSlashInput()
		printCmd := m.printTranscriptCmd(false)
		return m, printCmd, true
	}
	mode := hostcmd.SubshellModeLocalBash
	if m.Remote.Active {
		mode = hostcmd.SubshellModeRemoteSSH
	}
	if m.CommandSender != nil {
		msgs := append([]string(nil), m.TranscriptLines()...)
		hist := append([]string(nil), m.Interaction.inputHistory...)
		_ = m.CommandSender.Send(hostcmd.ShellSnapshot{Messages: msgs, InputHistory: hist, Mode: mode})
	}
	m.clearSlashInput()
	return m, tea.Quit, true
}
