package approvalview

import "delve-shell/internal/i18n"

type ChoiceOption struct {
	Num   int
	Label string
}

func ChoiceCount(hasPending bool, hasPendingSensitive bool) int {
	switch {
	case hasPending:
		return 3
	case hasPendingSensitive:
		return 3
	default:
		return 0
	}
}

func ChoiceOptions(lang string, hasPending bool, hasPendingSensitive bool) []ChoiceOption {
	switch {
	case hasPending:
		return []ChoiceOption{
			{1, i18n.T(lang, i18n.KeyChoiceApprove)},
			{2, i18n.T(lang, i18n.KeyChoiceDismiss)},
			{3, i18n.T(lang, i18n.KeyChoiceCopy)},
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

func InputPlaceholder(lang string, hasPending bool, hasPendingSensitive bool) string {
	switch {
	case hasPending:
		return i18n.T(lang, i18n.KeyInputHintApproveThree)
	case hasPendingSensitive:
		return i18n.T(lang, i18n.KeyInputHintSensitive)
	default:
		return i18n.T(lang, i18n.KeyPlaceholderInput)
	}
}
