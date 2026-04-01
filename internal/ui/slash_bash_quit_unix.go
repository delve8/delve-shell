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
		m.appendUserSubmittedEcho(text)
		m.AppendTranscriptLines(errStyle.Render(m.delveMsg(i18n.T(i18n.KeyOfflineExecBashDisabled))))
		m.clearSlashInput()
		printCmd := m.printTranscriptCmd(false)
		return m, printCmd, true
	}
	m.appendUserSubmittedEcho(text)
	mode := hostcmd.SubshellModeLocalBash
	if m.Remote.Active {
		mode = hostcmd.SubshellModeRemoteSSH
	}
	if m.CommandSender != nil {
		msgs := append([]string(nil), m.TranscriptLines()...)
		_ = m.CommandSender.Send(hostcmd.ShellSnapshot{Messages: msgs, Mode: mode})
	}
	m.clearSlashInput()
	return m, tea.Quit, true
}
