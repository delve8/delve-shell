package hostcontroller

import (
	"delve-shell/internal/hostbus"
	"delve-shell/internal/ui"
)

// handleSlashRelayToUI re-injects structured slash intent into the TUI via the UI message queue.
func (c *Controller) handleSlashRelayToUI(e hostbus.Event) {
	if e.SlashSubmit == nil {
		return
	}
	p := *e.SlashSubmit
	c.ui.Raw(ui.SlashSubmitRelayMsg{Payload: p})
}
