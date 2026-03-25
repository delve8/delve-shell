package controller

import (
	"delve-shell/internal/host/bus"
	"delve-shell/internal/ui"
)

// handleSlashRelayToUI re-injects structured slash intent into the TUI via the UI message queue.
func (c *Controller) handleSlashRelayToUI(e bus.Event) {
	if e.SlashSubmit == nil {
		return
	}
	p := *e.SlashSubmit
	c.ui.Raw(ui.SlashSubmitRelayMsg{
		RawLine:            p.RawLine,
		SlashSelectedIndex: p.SlashSelectedIndex,
		InputLine:          p.InputLine,
	})
}
