package ui

import (
	"delve-shell/internal/slashview"
	"delve-shell/internal/uiregistry"
)

// SlashOption is one row in the slash command list (command + description).
// It is a UI view-model; provider registries may use their own internal types.
type SlashOption struct {
	Cmd  string
	Desc string
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
