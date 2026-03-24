package ui

type uiState string

const (
	uiStateMainInput        uiState = "main_input"
	uiStatePendingApproval  uiState = "pending_approval"
	uiStatePendingSensitive uiState = "pending_sensitive"
	uiStateOverlay          uiState = "overlay"
)

// currentUIState is a lightweight FSM view of current UI mode.
// Priority follows interactive exclusivity: pending > overlay > main.
func (m Model) currentUIState() uiState {
	if m.PendingSensitive != nil {
		return uiStatePendingSensitive
	}
	if m.Pending != nil {
		return uiStatePendingApproval
	}
	if m.Overlay.Active {
		return uiStateOverlay
	}
	return uiStateMainInput
}
