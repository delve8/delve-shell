package hostcontroller

import "delve-shell/internal/hostbus"

// handleSlashEntered runs after the TUI has executed a slash handler. Business logic and tea.Cmd stay in the UI;
// the bus event exists for observability and future routing (metrics, staged migration). Do not execute commands here.
func (c *Controller) handleSlashEntered(e hostbus.Event) {
	_ = e.UserText
}
