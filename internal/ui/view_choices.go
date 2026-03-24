package ui

import "delve-shell/internal/i18n"

// choiceOption is one line in the choice menu (num 1-based, label for display).
type choiceOption struct {
	Num   int
	Label string
}

// choiceCount returns the number of options when in a choice state (approval 2 or 3, sensitive 3, or session list N).
func choiceCount(m Model) int {
	switch {
	case m.Pending != nil:
		if m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun() {
			return 3 // Run, Copy, Dismiss
		}
		return 2 // Run, Reject
	case m.PendingSensitive != nil:
		return 3
	default:
		return 0
	}
}

// getChoiceOptions returns the option list for the current choice state (approval 2 or 3 options / sensitive / session list).
func getChoiceOptions(m Model, lang string) []choiceOption {
	switch {
	case m.Pending != nil:
		if m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun() {
			return []choiceOption{
				{1, i18n.T(lang, i18n.KeyChoiceApprove)},
				{2, i18n.T(lang, i18n.KeyChoiceCopy)},
				{3, i18n.T(lang, i18n.KeyChoiceDismiss)},
			}
		}
		return []choiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceApprove)},
			{2, i18n.T(lang, i18n.KeyChoiceReject)},
		}
	case m.PendingSensitive != nil:
		return []choiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceRefuse)},
			{2, i18n.T(lang, i18n.KeyChoiceRunStore)},
			{3, i18n.T(lang, i18n.KeyChoiceRunNoStore)},
		}
	default:
		return nil
	}
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	lang := m.getLang()
	switch {
	case m.Pending != nil:
		if m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun() {
			m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintApproveThree)
		} else {
			m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintApprove)
		}
	case m.PendingSensitive != nil:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyInputHintSensitive)
	default:
		m.Input.Placeholder = i18n.T(lang, i18n.KeyPlaceholderInput)
	}
}
