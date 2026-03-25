package ui

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
	"delve-shell/internal/ui/widget"
)

// slashDropdownBelowInput returns extra lines to show under the input when in slash mode (not in approval/sensitive choice).
func (m Model) slashDropdownBelowInput(lang string) string {
	inputVal := m.Input.Value()
	if !strings.HasPrefix(inputVal, "/") {
		return ""
	}
	opts := getSlashOptionsForInput(inputVal, lang, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Host.RemoteActive())
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) == 0 {
		return ""
	}
	const maxSlashVisible = 4
	rows := slashview.BuildDropdownRows(toSlashViewOptions(opts), vis, m.Interaction.slashSuggestIndex, m.Layout.Width, maxSlashVisible)
	list := make([]widget.ListRow, len(rows))
	for i, row := range rows {
		list[i] = widget.ListRow{Text: row.Text, Highlight: row.Highlight}
	}
	return widget.RenderLinesBelowInput("   ", list, suggestStyle, suggestHi)
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
	lines := maininput.BuildChoiceLines(adapted, m.Interaction.ChoiceIndex)
	list := make([]widget.ListRow, len(lines))
	for i, line := range lines {
		list[i] = widget.ListRow{Text: line.Text, Highlight: line.Highlight}
	}
	return widget.RenderLinesBelowInput(" ", list, suggestStyle, suggestHi)
}

// waitingLineBelowInput returns the "wait or /cancel" hint when AI is running (empty if not applicable).
func (m Model) waitingLineBelowInput(lang string) string {
	inChoice := m.hasPendingApproval()
	return maininput.WaitingHint(m.Interaction.WaitingForAI, inChoice, suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel)))
}
