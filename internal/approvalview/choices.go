package approvalview

import "delve-shell/internal/i18n"

type ChoiceOption struct {
	Num   int
	Label string
}

func ChoiceCount(hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) int {
	switch {
	case hasPending:
		if !allowlistAutoRunEnabled {
			return 3
		}
		return 2
	case hasPendingSensitive:
		return 3
	default:
		return 0
	}
}

func ChoiceOptions(lang string, hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) []ChoiceOption {
	switch {
	case hasPending:
		if !allowlistAutoRunEnabled {
			return []ChoiceOption{
				{1, i18n.T(lang, i18n.KeyChoiceApprove)},
				{2, i18n.T(lang, i18n.KeyChoiceCopy)},
				{3, i18n.T(lang, i18n.KeyChoiceDismiss)},
			}
		}
		return []ChoiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceApprove)},
			{2, i18n.T(lang, i18n.KeyChoiceReject)},
		}
	case hasPendingSensitive:
		return []ChoiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceRefuse)},
			{2, i18n.T(lang, i18n.KeyChoiceRunStore)},
			{3, i18n.T(lang, i18n.KeyChoiceRunNoStore)},
		}
	default:
		return nil
	}
}

func InputPlaceholder(lang string, hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) string {
	switch {
	case hasPending:
		if !allowlistAutoRunEnabled {
			return i18n.T(lang, i18n.KeyInputHintApproveThree)
		}
		return i18n.T(lang, i18n.KeyInputHintApprove)
	case hasPendingSensitive:
		return i18n.T(lang, i18n.KeyInputHintSensitive)
	default:
		return i18n.T(lang, i18n.KeyPlaceholderInput)
	}
}
