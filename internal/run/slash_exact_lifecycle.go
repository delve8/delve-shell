package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashExact("/help", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return m.OpenHelpOverlay(), nil
		},
		ClearInput: true,
	})
	ui.RegisterSlashExact("/q", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) { return m, tea.Quit },
		ClearInput: false,
	})
	ui.RegisterSlashExact("/sh", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			if m.Ports.ShellRequestedChan != nil {
				msgs := make([]string, len(m.Messages))
				copy(msgs, m.Messages)
				select {
				case m.Ports.ShellRequestedChan <- msgs:
				default:
				}
			}
			return m, tea.Quit
		},
		ClearInput: false,
	})
}
