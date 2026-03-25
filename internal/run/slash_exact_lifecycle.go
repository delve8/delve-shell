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
	ui.RegisterSlashExact("/sh", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			_ = m.EmitShellSnapshotIntent(m.TranscriptLines())
			return m, tea.Quit
		},
		ClearInput: true,
	})
}
