//go:build !windows

package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
)

// trySlashBashQuit handles /bash: snapshot transcript and quit the TUI to spawn a shell.
func trySlashBashQuit(m Model, text string) (Model, tea.Cmd, bool) {
	if text != "/bash" {
		return m, nil, false
	}
	if m.offlineExecutionMode() {
		m = m.appendUserSubmittedEcho(text)
		m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(i18n.T(m.getLang(), i18n.KeyOfflineExecBashDisabled))))
		m = m.clearSlashInput()
		m2, printCmd := m.printTranscriptCmd(false)
		return m2, printCmd, true
	}
	m = m.appendUserSubmittedEcho(text)
	mode := hostcmd.SubshellModeLocalBash
	if m.Remote.Active {
		mode = hostcmd.SubshellModeRemoteSSH
	}
	_ = m.EmitShellSnapshotIntentWithMode(m.TranscriptLines(), mode)
	return m.clearSlashInput(), tea.Quit, true
}
