package ui

type uiState string

const (
	uiStateMainInput        uiState = "main_input"
	uiStateChoiceCard       uiState = "choice_card"
	uiStateChoiceCardAlt    uiState = "choice_card_alt"
	uiStateOverlay          uiState = "overlay"
)

// currentUIState is a lightweight FSM view of current UI mode.
// Priority follows interactive exclusivity: pending > overlay > main.
func (m Model) currentUIState() uiState {
	if m.ChoiceCard.pendingSensitive != nil {
		return uiStateChoiceCardAlt
	}
	if m.ChoiceCard.pending != nil {
		return uiStateChoiceCard
	}
	if m.Overlay.Active {
		return uiStateOverlay
	}
	return uiStateMainInput
}
