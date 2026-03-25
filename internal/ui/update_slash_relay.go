package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleSlashSubmitRelayMsg(msg SlashSubmitRelayMsg) (Model, tea.Cmd) {
	return m.executeMainEnterCommandNoRelay(msg.Payload.RawLine, msg.Payload.SlashSelectedIndex)
}
