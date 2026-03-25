package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashreg"
	"delve-shell/internal/slashview"
	"delve-shell/internal/ui/widget"
	"delve-shell/internal/uiregistry"
)

// SlashOption is one row in the slash command list (command + description).
// It is a UI view-model; provider registries may use their own internal types.
type SlashOption struct {
	Cmd  string
	Desc string
}

// SlashExactDispatchEntry defines an exact slash command handler.
// The registry is populated from feature packages via explicit Register() (see bootstrap.Install).
type SlashExactDispatchEntry struct {
	Handle     func(Model) (Model, tea.Cmd)
	ClearInput bool
}

// SlashPrefixDispatchEntry routes slash commands with arguments by prefix match.
// Registry is populated by feature packages' Register() (wired through bootstrap.Install).
type SlashPrefixDispatchEntry struct {
	Prefix string
	Handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

var slashExactDispatchRegistry = slashreg.NewExactRegistry[Model, tea.Cmd]()
var slashPrefixDispatchRegistry = slashreg.NewPrefixRegistry[Model, tea.Cmd]()

// RegisterSlashExact registers an exact slash command handler.
// Intended to be called from feature packages' Register() functions.
func RegisterSlashExact(cmd string, entry SlashExactDispatchEntry) {
	if cmd == "" {
		return
	}
	slashExactDispatchRegistry.Set(cmd, slashreg.ExactEntry[Model, tea.Cmd]{
		Handle:     entry.Handle,
		ClearInput: entry.ClearInput,
	})
}

// RegisterSlashPrefix registers a prefix-based slash command handler.
// Intended to be called from feature packages' Register() functions.
func RegisterSlashPrefix(prefix string, entry SlashPrefixDispatchEntry) {
	if prefix == "" {
		return
	}
	if entry.Prefix == "" {
		entry.Prefix = prefix
	}
	slashPrefixDispatchRegistry.Set(prefix, slashreg.PrefixEntry[Model, tea.Cmd]{
		Prefix: entry.Prefix,
		Handle: entry.Handle,
	})
}

// getSlashOptions returns top-level slash commands from registered providers.
func getSlashOptions(lang string) []SlashOption {
	raw := uiregistry.RootSlashOptions(lang)
	out := make([]SlashOption, 0, len(raw))
	for _, o := range raw {
		out = append(out, SlashOption{Cmd: o.Cmd, Desc: o.Desc})
	}
	return out
}

// getSlashOptionsForInput returns slash options to show.
// Specialized domains (e.g. /sessions, /run, /config) are expected to be handled by providers.
func getSlashOptionsForInput(inputVal string, lang string) []SlashOption {
	raw := uiregistry.SlashOptionsForInput(inputVal, lang)
	out := make([]SlashOption, 0, len(raw))
	for _, o := range raw {
		out = append(out, SlashOption{Cmd: o.Cmd, Desc: o.Desc})
	}
	return out
}

// visibleSlashOptions filters options by input prefix and returns matching indices.
func visibleSlashOptions(input string, opts []SlashOption) []int {
	return slashview.VisibleIndices(input, toSlashViewOptions(opts))
}

// slashChosenToInputValue converts the chosen slash command to the string to put in the input (strips <placeholder> and adds space).
func slashChosenToInputValue(chosen string) string {
	return slashview.ChosenToInputValue(chosen)
}

func toSlashViewOptions(opts []SlashOption) []slashview.Option {
	adapted := make([]slashview.Option, 0, len(opts))
	for _, opt := range opts {
		adapted = append(adapted, slashview.Option{Cmd: opt.Cmd, Desc: opt.Desc})
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
