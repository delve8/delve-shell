package ui

type uiState string

const (
	uiStateMainInput     uiState = "main_input"
	uiStateChoiceCard    uiState = "choice_card"
	uiStateChoiceCardAlt uiState = "choice_card_alt"
	uiStateOverlay       uiState = "overlay"
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

// TranscriptLines returns a copy of the current transcript lines shown in the main viewport.
func (m Model) TranscriptLines() []string {
	if len(m.messages) == 0 {
		return nil
	}
	out := make([]string, len(m.messages))
	copy(out, m.messages)
	return out
}

// WithTranscriptLines replaces the transcript with the provided lines (copied).
func (m Model) WithTranscriptLines(lines []string) Model {
	if len(lines) == 0 {
		m.messages = nil
		return m
	}
	out := make([]string, len(lines))
	copy(out, lines)
	m.messages = out
	return m
}

// AppendTranscriptLines appends rendered transcript lines.
func (m Model) AppendTranscriptLines(lines ...string) Model {
	if len(lines) == 0 {
		return m
	}
	m.messages = append(m.messages, lines...)
	return m
}
