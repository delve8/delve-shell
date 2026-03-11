package ui

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
)

const maxSessionsInSlash = 20

// slashOption is one row in the slash command list (command + description).
// Path is set only for /sessions items; when user selects such an option, switch to that session.
type slashOption struct {
	Cmd  string
	Desc string
	Path string // session file path when this option is a session to switch to
}

// getSlashOptions returns top-level slash commands (shown when input starts with "/"); order: help, cancel, config, new, sessions, reload, run, sh, exit.
func getSlashOptions(lang string) []slashOption {
	return []slashOption{
		{"/help", i18n.T(lang, i18n.KeyDescHelp), ""},
		{"/cancel", i18n.T(lang, i18n.KeyDescCancel), ""},
		{"/config", i18n.T(lang, i18n.KeyDescConfig), ""},
		{"/remote on", i18n.T(lang, i18n.KeyDescRemoteOn), ""},
		{"/remote off", i18n.T(lang, i18n.KeyDescRemoteOff), ""},
		{"/new", i18n.T(lang, i18n.KeySessionNew), ""},
		{"/sessions", i18n.T(lang, i18n.KeyDescSessions), ""},
		{"/reload", i18n.T(lang, i18n.KeyDescReload), ""},
		{"/run <cmd>", i18n.T(lang, i18n.KeyDescRun), ""},
		{"/sh", i18n.T(lang, i18n.KeyDescSh), ""},
		{"/exit", i18n.T(lang, i18n.KeyDescExit), ""},
		{"/q", i18n.T(lang, i18n.KeyDescExit), ""},
	}
}

// getConfigSubOptions returns /config sub-options (shown when input starts with "/config").
// Order: frequent first (remote, show, auto-run/allowlist), LLM last (set once at init, rarely changed).
func getConfigSubOptions(lang string) []slashOption {
	return []slashOption{
		{"/config add-remote", i18n.T(lang, i18n.KeyDescConfigAddRemote), ""},
		{"/config remove-remote", i18n.T(lang, i18n.KeyDescConfigRemoveRemote), ""},
		{"/config auto-run list-only", i18n.T(lang, i18n.KeyDescAutoRunListOnly), ""},
		{"/config auto-run disable", i18n.T(lang, i18n.KeyDescAutoRunDisable), ""},
		{"/config show", i18n.T(lang, i18n.KeyDescConfigShow), ""},
		{"/config update auto-run list", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate), ""},
		{"/config llm base_url <url>", i18n.T(lang, i18n.KeyDescConfigLLMBaseURL), ""},
		{"/config llm api_key <key>", i18n.T(lang, i18n.KeyDescConfigLLMApiKey), ""},
		{"/config llm model <name>", i18n.T(lang, i18n.KeyDescConfigLLMModel), ""},
	}
}

// getSlashOptionsForInput returns slash options to show: when input is "/config" or "/config xxx" returns only /config sub-options; when "/sessions" or "/sessions xxx" returns session list (with Path set) for switch, excluding currentSessionPath so first option is another session; else top-level commands.
func getSlashOptionsForInput(inputVal string, lang string, currentSessionPath string) []slashOption {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)
	if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
		if rest == "remove-remote" || strings.HasPrefix(rest, "remove-remote ") {
			filter := strings.TrimSpace(strings.TrimPrefix(rest, "remove-remote"))
			return getRemoveRemoteSlashOptions(lang, filter)
		}
		return getConfigSubOptions(lang)
	}
	if normalizedLower == "sessions" || strings.HasPrefix(normalizedLower, "sessions ") {
		// Trim "sessions" (no trailing space) so "/sessions" yields filter "" and shows all sessions
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "sessions"))
		return getSessionSlashOptions(filter, currentSessionPath)
	}
	if normalizedLower == "remote" || strings.HasPrefix(normalizedLower, "remote ") {
		// Filter is what user typed after "remote on " (e.g. "/remote on" -> "", "/remote on dev" -> "dev"). Do not use "on" as filter.
		filter := ""
		if strings.HasPrefix(normalizedLower, "remote on") {
			filter = strings.TrimSpace(strings.TrimPrefix(normalizedLower, "remote on"))
		}
		return getRemoteSlashOptions(filter, lang)
	}
	return getSlashOptions(lang)
}

// getRemoteSlashOptions returns slash options for remote connection; filter is the substring after "/remote ".
// Shows configured remotes first, then manual input option.
func getRemoteSlashOptions(filter string, lang string) []slashOption {
	var opts []slashOption
	remotes, err := config.LoadRemotes()
	if err == nil && len(remotes) > 0 {
		filterLower := strings.ToLower(filter)
		for _, r := range remotes {
			line := r.Target + " " + r.Name
			if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
				continue
			}
			// When no name, Cmd already shows target; leave Desc empty to avoid duplication.
			desc := r.Name
			opts = append(opts, slashOption{
				Cmd:  "/remote on " + config.HostFromTarget(r.Target),
				Desc: desc,
			})
		}
	}
	opts = append(opts, slashOption{Cmd: "/remote on <user@host>", Desc: i18n.T(lang, i18n.KeyRemoteManualHint), Path: ""})
	return opts
}

// getRemoveRemoteSlashOptions returns slash options for /config remove-remote: one option per configured remote (select to remove).
func getRemoveRemoteSlashOptions(lang string, filter string) []slashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []slashOption{{Cmd: "/config remove-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}
	filterLower := strings.ToLower(filter)
	var opts []slashOption
	for _, r := range remotes {
		line := r.Target + " " + r.Name
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}
		// Show host only (no username). RemoveRemoteByName accepts host and matches by HostFromTarget(r.Target).
		desc := r.Name
		opts = append(opts, slashOption{
			Cmd:  "/config remove-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
		})
	}
	if len(opts) == 0 {
		return []slashOption{{Cmd: "/config remove-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}
	return opts
}

// getSessionSlashOptions returns slash options for session list; filter is the substring after "/sessions " (e.g. date or time to filter).
// currentSessionPath is excluded so the first option is always another session (avoids "switch" loading same session).
func getSessionSlashOptions(filter string, currentSessionPath string) []slashOption {
	summaries, err := history.ListSessionsWithSummary(maxSessionsInSlash)
	if err != nil || len(summaries) == 0 {
		return []slashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: "", Path: ""}}
	}
	filterLower := strings.ToLower(filter)
	var opts []slashOption
	for _, s := range summaries {
		if s.Path == currentSessionPath {
			continue
		}
		if filterLower != "" {
			line := s.ID
			if s.Snippet != "" {
				line += " " + s.Snippet
			}
			if !strings.Contains(strings.ToLower(line), filterLower) {
				continue
			}
		}
		// Use session id as primary (stable); mtime changes on write.
		cmd := s.ID
		if s.Snippet != "" {
			cmd += "  " + s.Snippet
		}
		opts = append(opts, slashOption{Cmd: cmd, Desc: "", Path: s.Path})
	}
	if len(opts) == 0 {
		return []slashOption{{Cmd: i18n.T("en", i18n.KeySessionNone), Desc: "", Path: ""}} // Path empty = no session to switch to
	}
	return opts
}

// visibleSlashOptions filters options by input prefix and returns matching indices.
// For session options (Path != ""), the input part after "/sessions " filters by substring match on Cmd.
func visibleSlashOptions(input string, opts []slashOption) []int {
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimSpace(input)
	inputLower := strings.ToLower(input)
	// Session list: options have Path set; filter by substring after "sessions "
	if len(opts) > 0 && opts[0].Path != "" {
		// All opts are sessions; filter already applied in getSessionSlashOptions
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
