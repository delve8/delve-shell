package ui

import (
	"delve-shell/internal/slashview"
	"delve-shell/internal/uiregistry"
	"delve-shell/internal/uitypes"
)

// SlashOption is the exported view-model row for slash command suggestions.
type SlashOption = uitypes.SlashOption

// SlashRunUsageOption is the Cmd string for the /run usage row in slash suggestions (fill-only on select).
const SlashRunUsageOption = uitypes.SlashRunUsageOption

// getSlashOptions returns top-level slash commands from registered providers.
func getSlashOptions(lang string) []SlashOption {
	return uiregistry.RootSlashOptions(lang)
}

// getSlashOptionsForInput returns slash options to show.
// Specialized domains (e.g. /sessions, /run, /config) are expected to be handled by providers.
func getSlashOptionsForInput(inputVal string, lang string, localRunCommands []string, remoteRunCommands []string, remoteActive bool) []SlashOption {
	return uiregistry.SlashOptionsForInput(inputVal, lang, localRunCommands, remoteRunCommands, remoteActive)
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
