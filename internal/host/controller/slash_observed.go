package controller

import "delve-shell/internal/host/bus"

// handleSlashEntered runs after the TUI has executed a slash handler. Business logic and tea.Cmd stay in the UI;
// the bus event exists for observability and future routing (metrics, staged migration). Do not execute commands here.
func (c *Controller) handleSlashEntered(e bus.Event) {
	_ = e.UserText
}
