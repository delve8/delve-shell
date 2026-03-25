package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func registerSlashExactCancelCmd() {
	ui.RegisterSlashExact("/cancel", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Interaction.WaitingForAI {
				_ = m.EmitCancelRequestIntent()
				m.Interaction.WaitingForAI = false
				return m, nil
			}
			m = m.AppendTranscriptLines(delveMsg("en", i18n.T("en", i18n.KeyNoRequestInProgress)))
			return m.RefreshViewport(), nil
		},
		ClearInput: true,
	})
}
