package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashdispatch"
	"delve-shell/internal/slashview"
	"delve-shell/internal/ui/widget"
	"delve-shell/internal/uiregistry"
)

// SlashOption is one row in the slash command list (command + description).
// It is a UI view-model; provider registries may use their own internal types.
type SlashOption struct {
	Cmd       string
	Desc      string
	FillValue string
}

var slashRuntime = slashdispatch.NewRuntime[Model, tea.Cmd]()

// getSlashOptionsForInput returns slash options to show.
// Specialized domains (e.g. /sessions, /run, /config) are expected to be handled by providers.
func getSlashOptionsForInput(inputVal string, lang string) []SlashOption {
	raw := uiregistry.SlashOptionsForInput(inputVal, lang)
	out := make([]SlashOption, 0, len(raw))
	for _, o := range raw {
		out = append(out, SlashOption{Cmd: o.Cmd, Desc: o.Desc, FillValue: o.FillValue})
	}
	return out
}

// visibleSlashOptions filters options by input prefix and returns matching indices.
func visibleSlashOptions(input string, opts []SlashOption) []int {
	return slashview.VisibleIndices(input, toSlashViewOptions(opts))
}

func toSlashViewOptions(opts []SlashOption) []slashview.Option {
	adapted := make([]slashview.Option, 0, len(opts))
	for _, opt := range opts {
		adapted = append(adapted, slashview.Option{Cmd: opt.Cmd, Desc: opt.Desc, FillValue: opt.FillValue})
	}
	return adapted
}

// slashSuggestionContext returns options, visible indices, and slashview rows for the current input buffer
// using Model.getLang. Use slashSuggestionContextWithLang when the view passes an explicit lang.
func (m Model) slashSuggestionContext(inputVal string) (opts []SlashOption, vis []int, viewOpts []slashview.Option) {
	return m.slashSuggestionContextWithLang(inputVal, m.getLang())
}

// slashSuggestionContextWithLang is the same as slashSuggestionContext but uses an explicit UI language.
func (m Model) slashSuggestionContextWithLang(inputVal, lang string) (opts []SlashOption, vis []int, viewOpts []slashview.Option) {
	opts = getSlashOptionsForInput(inputVal, lang)
	vis = visibleSlashOptions(inputVal, opts)
	viewOpts = toSlashViewOptions(opts)
	return opts, vis, viewOpts
}

// slashDropdownBelowInput returns extra lines to show under the input when in slash mode (not in approval/sensitive choice).
func (m Model) slashDropdownBelowInput(lang string) string {
	inputVal := m.Input.Value()
	if !strings.HasPrefix(inputVal, "/") {
		return ""
	}
	_, vis, viewOpts := m.slashSuggestionContextWithLang(inputVal, lang)
	if len(vis) == 0 {
		return ""
	}
	const maxSlashVisible = 4
	rows := slashview.BuildDropdownRows(viewOpts, vis, m.Interaction.slashSuggestIndex, m.layout.Width, maxSlashVisible)
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

// waitingLineBelowInput returns the "wait or press Esc to cancel" hint when AI is running.
func (m Model) waitingLineBelowInput(lang string) string {
	inChoice := m.hasPendingChoiceCard()
	return maininput.WaitingHint(m.Interaction.WaitingForAI, inChoice, suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel)))
}
