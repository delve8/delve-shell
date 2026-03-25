package ui

import (
	"delve-shell/internal/uiflow/choicecard"
)

// choiceOption is one line in the choice menu (num 1-based, label for display).
type choiceOption struct {
	Num   int
	Label string
}

// choiceCount returns the number of options when in a choice state (approval 2 or 3, sensitive 3, or session list N).
func choiceCount(m Model) int {
	allowlistAutoRunEnabled := m.Host.AllowlistAutoRunEnabled()
	return choicecard.ChoiceCount(m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
}

// getChoiceOptions returns the option list for the current choice state (approval 2 or 3 options / sensitive / session list).
func getChoiceOptions(m Model, lang string) []choiceOption {
	allowlistAutoRunEnabled := m.Host.AllowlistAutoRunEnabled()
	opts := choicecard.ChoiceOptions(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
	out := make([]choiceOption, 0, len(opts))
	for _, opt := range opts {
		out = append(out, choiceOption{Num: opt.Num, Label: opt.Label})
	}
	return out
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	lang := m.getLang()
	allowlistAutoRunEnabled := m.Host.AllowlistAutoRunEnabled()
	m.Input.Placeholder = choicecard.InputPlaceholder(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
}
