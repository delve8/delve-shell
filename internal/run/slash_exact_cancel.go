package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostapp"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashExact("/cancel", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Interaction.WaitingForAI {
				_ = hostapp.PublishCancelRequest()
				m.Interaction.WaitingForAI = false
				return m, nil
			}
			m.Messages = append(m.Messages, delveMsg("en", i18n.T("en", i18n.KeyNoRequestInProgress)))
			return m.RefreshViewport(), nil
		},
		ClearInput: true,
	})
}
