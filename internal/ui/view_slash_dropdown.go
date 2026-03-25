package ui

import (
	"strings"

	"delve-shell/internal/hostapp"
	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
)

// slashDropdownBelowInput returns extra lines to show under the input when in slash mode (not in approval/sensitive choice).
func (m Model) slashDropdownBelowInput(lang string) string {
	inputVal := m.Input.Value()
	if !strings.HasPrefix(inputVal, "/") {
		return ""
	}
	opts := getSlashOptionsForInput(inputVal, lang, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, hostapp.RemoteActive())
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) == 0 {
		return ""
	}
	var out strings.Builder
	out.WriteString("\n")
	const maxSlashVisible = 4
	rows := slashview.BuildDropdownRows(toSlashViewOptions(opts), vis, m.Interaction.slashSuggestIndex, m.Layout.Width, maxSlashVisible)
	for _, row := range rows {
		if row.Highlight {
			out.WriteString(suggestHi.Render("   "+row.Text) + "\n")
		} else {
			out.WriteString(suggestStyle.Render("   "+row.Text) + "\n")
		}
	}
	return out.String()
}

// choiceLinesBelowInput returns extra lines for numeric choice menu under the input.
func (m Model) choiceLinesBelowInput(lang string) string {
	opts := getChoiceOptions(m, lang)
	if len(opts) == 0 {
		return ""
	}
	adapted := make([]maininput.ChoiceOption, 0, len(opts))
	for _, o := range opts {
		adapted = append(adapted, maininput.ChoiceOption{Num: o.Num, Label: o.Label})
	}
	var out strings.Builder
	out.WriteString("\n")
	for _, line := range maininput.BuildChoiceLines(adapted, m.Interaction.ChoiceIndex) {
		if line.Highlight {
			out.WriteString(suggestHi.Render(" "+line.Text) + "\n")
		} else {
			out.WriteString(suggestStyle.Render(" "+line.Text) + "\n")
		}
	}
	return out.String()
}

// waitingLineBelowInput returns the "wait or /cancel" hint when AI is running (empty if not applicable).
func (m Model) waitingLineBelowInput(lang string) string {
	inChoice := m.hasPendingApproval()
	return maininput.WaitingHint(m.Interaction.WaitingForAI, inChoice, suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel)))
}
