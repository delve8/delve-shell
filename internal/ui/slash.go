package ui

import (
	"strings"
)

// slashOption is one row in the slash command list (command + description).
// Path is set only for /sessions items; when user selects such an option, switch to that session.
type slashOption struct {
	Cmd  string
	Desc string
	Path string // session file path when this option is a session to switch to
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
func getSlashOptionsForInput(inputVal string, lang string, currentSessionPath string, localRunCommands []string, remoteRunCommands []string, remoteActive bool) []SlashOption {
	for _, p := range slashOptionsProviderChain.List() {
		if opts, handled := p(inputVal, lang, currentSessionPath, localRunCommands, remoteRunCommands, remoteActive); handled {
			return opts
		}
	}
	return getSlashOptions(lang)
}

// visibleSlashOptions filters options by input prefix and returns matching indices.
// For session options (Path != ""), the input part after "/sessions " filters by substring match on Cmd.
func visibleSlashOptions(input string, opts []SlashOption) []int {
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimSpace(input)
	inputLower := strings.ToLower(input)
	// Session list: options have Path set; filter by substring after "sessions "
	if len(opts) > 0 && opts[0].Path != "" {
		// All opts are sessions; filtering is already done by provider side.
		out := make([]int, len(opts))
		for i := range opts {
			out[i] = i
		}
		return out
	}
	if len(opts) == 1 && opts[0].Path == "" {
		return []int{0}
	}
	var out []int
	for i, opt := range opts {
		base := strings.Split(opt.Cmd, " ")[0]
		base = strings.TrimPrefix(base, "/")
		if inputLower == "" || strings.HasPrefix(base, inputLower) || strings.HasPrefix(opt.Cmd, "/"+inputLower) {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		for i := range opts {
			out = append(out, i)
		}
	}
	return out
}

// slashChosenToInputValue converts the chosen slash command to the string to put in the input (strips <placeholder> and adds space).
func slashChosenToInputValue(chosen string) string {
	if strings.Contains(chosen, " <") {
		if i := strings.Index(chosen, " <"); i > 0 {
			return chosen[:i] + " "
		}
	}
	return chosen
}
