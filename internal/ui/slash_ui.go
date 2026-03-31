package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/approvalview"
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

const inputBelowReserveRows = 4

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
	raw := uiregistry.SlashOptionsForInput(inputVal, lang)
	opts = make([]SlashOption, 0, len(raw))
	for _, o := range raw {
		opts = append(opts, SlashOption{Cmd: o.Cmd, Desc: o.Desc, FillValue: o.FillValue})
	}
	vis = visibleSlashOptions(inputVal, opts)
	viewOpts = toSlashViewOptions(opts)
	return opts, vis, viewOpts
}

// waitingLineText returns the waiting hint text without layout padding.
func (m Model) waitingLineText(lang string) string {
	inChoice := m.hasPendingChoiceCard()
	if m.Interaction.WaitingForAI && !inChoice {
		return suggestStyle.Render(i18n.T(lang, i18n.KeyWaitOrCancel))
	}
	return ""
}

// inputBelowBlock reserves the fixed-height block below the input so the footer position stays stable.
func (m Model) inputBelowBlock(lang string, inChoice bool) string {
	// Multiline: skip choice list / slash / fixed block unless walking input history (need hint + layout).
	if m.Input.LineCount() > 1 && !inChoice && m.Interaction.inputHistIndex < 0 {
		if m.Interaction.WaitingForAI {
			text := m.waitingLineText(lang)
			if text == "" {
				return "\n"
			}
			return "\n" + text + "\n"
		}
		return "\n"
	}
	rows := make([]widget.ListRow, 0, inputBelowReserveRows)
	reserveRows := inputBelowStableRows
	if inChoice {
		if m.ChoiceCard.offlinePaste != nil {
			hint := suggestStyle.Render(i18n.T(lang, i18n.KeyOfflinePasteHint))
			rows = []widget.ListRow{{Text: hint, PreRendered: true}}
		} else {
			opts := approvalview.ChoiceOptions(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil)
			adapted := make([]maininput.ChoiceOption, 0, len(opts))
			for _, o := range opts {
				adapted = append(adapted, maininput.ChoiceOption{Num: o.Num, Label: o.Label})
			}
			lines := maininput.BuildChoiceLines(adapted, m.Interaction.ChoiceIndex)
			rows = make([]widget.ListRow, len(lines))
			for i, line := range lines {
				rows[i] = widget.ListRow{Text: line.Text, Highlight: line.Highlight}
			}
		}
	} else if m.Interaction.inputHistIndex >= 0 {
		hint := i18n.T(lang, i18n.KeyInputHistBrowsingHint)
		styled := inputHistBrowsingHintStyle.Render("   — " + hint)
		rows = []widget.ListRow{{Text: styled, PreRendered: true}}
	} else if strings.HasPrefix(m.Input.Value(), "/") {
		_, vis, viewOpts := m.slashSuggestionContextWithLang(m.Input.Value(), lang)
		if len(vis) > 0 {
			const maxSlashVisible = inputBelowReserveRows
			rowsRaw := slashview.BuildDropdownRows(viewOpts, vis, m.Interaction.slashSuggestIndex, m.layout.Width, maxSlashVisible)
			rows = make([]widget.ListRow, len(rowsRaw))
			for i, row := range rowsRaw {
				rows[i] = widget.ListRow{Text: row.Text, Highlight: row.Highlight}
			}
		}
	}
	block := widget.RenderFixedLinesBelowInput("   ", rows, reserveRows, suggestStyle, suggestHi)
	if m.Interaction.WaitingForAI && !inChoice && !strings.HasPrefix(m.Input.Value(), "/") {
		waiting := i18n.T(lang, i18n.KeyWaitOrCancel)
		block = widget.RenderFixedLinesBelowInput("   ", []widget.ListRow{{Text: waiting}}, reserveRows, suggestStyle, suggestHi)
	}
	return block
}
