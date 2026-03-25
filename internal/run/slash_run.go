// Package run registers /run slash prefix and fill-only selection for the run usage option row.
// Black-box UI tests call [bootstrap.Install] (which imports this package) instead of mirroring handlers.
package run

import (
	"delve-shell/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func registerSlashRunCore() {
	ui.RegisterSlashSelectedProvider(func(m ui.Model, chosen string) (ui.Model, tea.Cmd, bool) {
		if chosen != slashRunUsageOption {
			return m, nil, false
		}
		m.Input.SetValue("/run ")
		m.Input.CursorEnd()
		return m, nil, true
	})
}
