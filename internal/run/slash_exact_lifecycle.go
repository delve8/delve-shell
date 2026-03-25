package run

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func registerSlashExactLifecycleCmds() {
	ui.RegisterSlashExact("/help", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return m.OpenHelpOverlay(), nil
		},
		ClearInput: true,
	})
	ui.RegisterSlashExact("/q", ui.SlashExactDispatchEntry{
		Handle:     func(m ui.Model) (ui.Model, tea.Cmd) { return m, tea.Quit },
		ClearInput: true,
	})
	ui.RegisterSlashExact("/sh", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			msgs := make([]string, len(m.Messages))
			copy(msgs, m.Messages)
			_ = m.Host.PublishShellSnapshot(msgs)
			return m, tea.Quit
		},
		ClearInput: true,
	})
}
