package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSlashSubmitRelayMsg(msg SlashSubmitRelayMsg) (Model, tea.Cmd) {
	if msg.Payload.InputLine != "" {
		m2, cmd, handled := m.execSlashEnterKeyLocal(msg.Payload.InputLine)
		if handled {
			return m2, cmd
		}
		return m.executeMainEnterCommandNoRelay(strings.TrimSpace(msg.Payload.InputLine), msg.Payload.SlashSelectedIndex)
	}
	return m.executeMainEnterCommandNoRelay(msg.Payload.RawLine, msg.Payload.SlashSelectedIndex)
}
