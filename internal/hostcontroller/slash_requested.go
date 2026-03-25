package hostcontroller

import "delve-shell/internal/hostbus"

// handleSlashRequested runs immediately before the TUI executes a matched slash handler.
// Business logic and tea.Cmd stay in the UI; the bus event exists for observability and future policy (metrics, staged migration).
func (c *Controller) handleSlashRequested(e hostbus.Event) {
	_ = e.UserText
}
