package ui

// WaitingForAI reports whether non-slash Enter submits are blocked while an LLM turn is in flight.
func (m Model) WaitingForAI() bool {
	return m.Interaction.WaitingForAI
}

// WithWaitingForAI returns a copy with WaitingForAI set (Bubble Tea style).
func (m Model) WithWaitingForAI(v bool) Model {
	m.Interaction.WaitingForAI = v
	return m
}

// SlashSuggestIndex returns the highlighted slash dropdown index (0-based).
func (m Model) SlashSuggestIndex() int {
	return m.Interaction.slashSuggestIndex
}

// WithSlashSuggestIndex returns a copy with slash suggestion index set.
func (m Model) WithSlashSuggestIndex(i int) Model {
	m.Interaction.slashSuggestIndex = i
	return m
}
