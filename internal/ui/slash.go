package ui

import (
	"strings"

	"delve-shell/internal/i18n"
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

// getSlashOptions returns top-level slash commands (shown when input starts with "/"); order: help, cancel, config, remote, new, sessions, skill, run, sh, quit.
func getSlashOptions(lang string) []SlashOption {
	return []slashOption{
		{"/help", i18n.T(lang, i18n.KeyDescHelp), ""},
		{"/cancel", i18n.T(lang, i18n.KeyDescCancel), ""},
		{"/config", i18n.T(lang, i18n.KeyDescConfig), ""},
		{"/remote", i18n.T(lang, i18n.KeyDescRemoteOn), ""},
		{"/new", i18n.T(lang, i18n.KeySessionNew), ""},
		{"/sessions", i18n.T(lang, i18n.KeyDescSessions), ""},
		{"/skill <skill-name> [detail]", i18n.T(lang, i18n.KeyDescSkill), ""},
		{"/run <cmd>", i18n.T(lang, i18n.KeyDescRun), ""},
		{"/sh", i18n.T(lang, i18n.KeyDescSh), ""},
		{"/q", i18n.T(lang, i18n.KeyDescExit), ""},
	}
}

// getConfigSubOptions returns /config sub-options (shown when input starts with "/config").
// Order: frequent first (remote, auto-run/allowlist, LLM), reload last.
func getConfigSubOptions(lang string) []SlashOption {
	return []slashOption{
		{"/config add-remote", i18n.T(lang, i18n.KeyDescConfigAddRemote), ""},
		{"/config del-remote", i18n.T(lang, i18n.KeyDescConfigRemoveRemote), ""},
		{"/config add-skill", i18n.T(lang, i18n.KeyDescSkillInstall), ""},
		{"/config del-skill", i18n.T(lang, i18n.KeyDescSkillRemove), ""},
		{"/config update-skill", i18n.T(lang, i18n.KeyDescConfigUpdateSkill), ""},
		{"/config auto-run list-only", i18n.T(lang, i18n.KeyDescAutoRunListOnly), ""},
		{"/config auto-run disable", i18n.T(lang, i18n.KeyDescAutoRunDisable), ""},
		{"/config update auto-run list", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate), ""},
		{"/config llm", i18n.T(lang, i18n.KeyDescConfigLLM), ""},
		{"/config reload", i18n.T(lang, i18n.KeyDescReload), ""},
	}
}

// getSlashOptionsForInput returns slash options to show: when input is "/config" or "/config xxx" returns only /config sub-options; when "/sessions" or "/sessions xxx" returns session list (with Path set) for switch, excluding currentSessionPath so first option is another session; else top-level commands.
func getSlashOptionsForInput(inputVal string, lang string, currentSessionPath string, localRunCommands []string, remoteRunCommands []string, remoteActive bool) []SlashOption {
	for _, p := range slashOptionsProviders {
		if opts, handled := p(inputVal, lang, currentSessionPath, localRunCommands, remoteRunCommands, remoteActive); handled {
			return opts
		}
	}

	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)
	// /run completion: show command candidates rather than slash commands.
	if normalizedLower == "run" || strings.HasPrefix(normalizedLower, "run ") {
		// When just "/run": show the usage form.
		if normalizedLower == "run" {
			return []slashOption{{Cmd: "/run <cmd>", Desc: i18n.T(lang, i18n.KeyDescRun), Path: ""}}
		}
		// After "/run " start showing command candidates.
		rest := ""
		if len(normalized) >= 3 {
			rest = strings.TrimSpace(normalized[3:])
		}
		// Only complete the first token after /run; once user has typed arguments, stop dropdown.
		if strings.Contains(rest, " ") || strings.Contains(rest, "\t") {
			return []slashOption{}
		}
		prefix := strings.ToLower(rest)
		cands := localRunCommands
		if cands == nil {
			cands = LocalRunCommands()
		}
		if remoteActive && len(remoteRunCommands) > 0 {
			cands = remoteRunCommands
		}
		// Limit suggestions to keep UI responsive and dropdown small.
		const maxRunCands = 50
		opts := make([]slashOption, 0, 12)
		for _, c := range cands {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(c), prefix) {
				continue
			}
			opts = append(opts, slashOption{Cmd: "/run " + c, Desc: "", Path: ""})
			if len(opts) >= maxRunCands {
				break
			}
		}
		// When no match, show nothing (do not fall back to top-level slash list).
		return opts
	}
	if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
		return getConfigSubOptions(lang)
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
