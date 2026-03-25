package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSlashSubmitRelayMsg(msg SlashSubmitRelayMsg) (Model, tea.Cmd) {
	if msg.InputLine != "" {
		m2, cmd, handled := m.execSlashEnterKeyLocal(msg.InputLine)
		if handled {
			return m2, cmd
		}
		return m.executeMainEnterCommandNoRelay(strings.TrimSpace(msg.InputLine), msg.SlashSelectedIndex)
	}
	return m.executeMainEnterCommandNoRelay(msg.RawLine, msg.SlashSelectedIndex)
}
