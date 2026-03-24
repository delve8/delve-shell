package ui

import (
	"delve-shell/internal/slashview"
)

// slashOption is one row in the slash command list (command + description).
type slashOption struct {
	Cmd  string
	Desc string
}

// SlashOption is the exported view-model row for slash command suggestions.
// It is an alias to the internal type so feature packages can build providers
// without being able to reference the unexported name.
type SlashOption = slashOption

// SlashRunUsageOption is the Cmd string for the /run usage row in slash suggestions (fill-only on select).
const SlashRunUsageOption = "/run <cmd>"

// getSlashOptions returns top-level slash commands from registered providers.
func getSlashOptions(lang string) []SlashOption {
	opts := make([]SlashOption, 0, 16)
	for _, p := range rootSlashOptionProviderChain.List() {
		opts = append(opts, p(lang)...)
	}
	return opts
}

// getSlashOptionsForInput returns slash options to show.
// Specialized domains (e.g. /sessions, /run, /config) are expected to be handled by providers.
func getSlashOptionsForInput(inputVal string, lang string, localRunCommands []string, remoteRunCommands []string, remoteActive bool) []SlashOption {
	for _, p := range slashOptionsProviderChain.List() {
		if opts, handled := p(inputVal, lang, localRunCommands, remoteRunCommands, remoteActive); handled {
			return opts
		}
	}
	return getSlashOptions(lang)
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
