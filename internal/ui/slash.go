package ui

import (
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/skills"
)

const maxSessionsInSlash = 20

// slashOption is one row in the slash command list (command + description).
// Path is set only for /sessions items; when user selects such an option, switch to that session.
type slashOption struct {
	Cmd  string
	Desc string
	Path string // session file path when this option is a session to switch to
}

// getSlashOptions returns top-level slash commands (shown when input starts with "/"); order: help, cancel, config, remote, new, sessions, skill, run, sh, quit.
func getSlashOptions(lang string) []slashOption {
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
func getConfigSubOptions(lang string) []slashOption {
	return []slashOption{
		{"/config add-remote", i18n.T(lang, i18n.KeyDescConfigAddRemote), ""},
		{"/config del-remote", i18n.T(lang, i18n.KeyDescConfigRemoveRemote), ""},
		{"/config add-skill", i18n.T(lang, i18n.KeyDescSkillInstall), ""},
		{"/config del-skill", i18n.T(lang, i18n.KeyDescSkillRemove), ""},
		{"/config auto-run list-only", i18n.T(lang, i18n.KeyDescAutoRunListOnly), ""},
		{"/config auto-run disable", i18n.T(lang, i18n.KeyDescAutoRunDisable), ""},
		{"/config update auto-run list", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate), ""},
		{"/config llm", i18n.T(lang, i18n.KeyDescConfigLLM), ""},
		{"/config reload", i18n.T(lang, i18n.KeyDescReload), ""},
	}
}

// getSlashOptionsForInput returns slash options to show: when input is "/config" or "/config xxx" returns only /config sub-options; when "/sessions" or "/sessions xxx" returns session list (with Path set) for switch, excluding currentSessionPath so first option is another session; else top-level commands.
func getSlashOptionsForInput(inputVal string, lang string, currentSessionPath string, localRunCommands []string, remoteRunCommands []string, remoteActive bool) []slashOption {
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
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
		if rest == "del-remote" || strings.HasPrefix(rest, "del-remote ") {
			filter := strings.TrimSpace(strings.TrimPrefix(rest, "del-remote"))
			return getRemoveRemoteSlashOptions(lang, filter)
		}
		if rest == "del-skill" || strings.HasPrefix(rest, "del-skill ") {
			filter := strings.TrimSpace(strings.TrimPrefix(rest, "del-skill"))
			return getDelSkillSlashOptions(lang, filter)
		}
		return getConfigSubOptions(lang)
	}
	if normalizedLower == "sessions" || strings.HasPrefix(normalizedLower, "sessions ") {
		// Trim "sessions" (no trailing space) so "/sessions" yields filter "" and shows all sessions
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "sessions"))
		return getSessionSlashOptions(filter, currentSessionPath)
	}
	if strings.TrimSpace(normalizedLower) == "remote" {
		// Only "/remote" (no "on"): show just /remote on and /remote off.
		return []slashOption{
			{"/remote on", i18n.T(lang, i18n.KeyDescRemoteOn), ""},
			{"/remote off", i18n.T(lang, i18n.KeyDescRemoteOff), ""},
		}
	}
	if strings.HasPrefix(normalizedLower, "remote on") {
		// "/remote on" or "/remote on xxx": show /remote off plus remote targets (and manual user@host).
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "remote on"))
		opts := getRemoteSlashOptions(filter, lang)
		offOpt := slashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff), Path: ""}
		return append([]slashOption{offOpt}, opts...)
	}
	if normalizedLower == "skill" || strings.HasPrefix(normalizedLower, "skill ") {
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "skill"))
		return getSkillSlashOptions(lang, filter)
	}
	return getSlashOptions(lang)
}

// getDelSkillSlashOptions returns options for /config del-skill: one option per installed skill.
func getDelSkillSlashOptions(lang string, filter string) []slashOption {
	list, err := skills.List()
	if err != nil || len(list) == 0 {
		return []slashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
	}
	filterLower := strings.ToLower(filter)
	var opts []slashOption
	for _, s := range list {
		if filterLower != "" && !strings.Contains(strings.ToLower(s.Name), filterLower) {
			continue
		}
		desc := strings.TrimSpace(s.Description)
		if desc == "" {
			desc = s.Name
		}
		cmdName := s.LocalName
		if cmdName == "" {
			cmdName = s.Name
		}
		opts = append(opts, slashOption{Cmd: "/config del-skill " + cmdName, Desc: desc, Path: ""})
	}
	if len(opts) == 0 {
		return []slashOption{{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeySkillNone), Path: ""}}
	}
	return opts
}

// getSkillSlashOptions returns options for /skill: list skills only. After user picks a skill they type natural language (no script list).
func getSkillSlashOptions(lang string, filter string) []slashOption {
	list, _ := skills.List()
	parts := strings.Fields(filter)
	if len(parts) == 0 {
		if len(list) == 0 {
			return []slashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: "", Path: ""}}
		}
		opts := make([]slashOption, 0, len(list))
		for _, s := range list {
			cmdName := s.LocalName
			if cmdName == "" {
				cmdName = s.Name
			}
			opts = append(opts, slashOption{Cmd: "/skill " + cmdName, Desc: s.Description, Path: ""})
		}
		return opts
	}
	skillName := parts[0]
	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		// No such skill: show skills whose name contains filter
		opts := make([]slashOption, 0)
		filterLower := strings.ToLower(skillName)
		for _, s := range list {
			if strings.Contains(strings.ToLower(s.Name), filterLower) {
				cmdName := s.LocalName
				if cmdName == "" {
					cmdName = s.Name
				}
				opts = append(opts, slashOption{Cmd: "/skill " + cmdName, Desc: s.Description, Path: ""})
			}
		}
		if len(opts) == 0 && len(list) > 0 {
			return opts
		}
		if len(opts) == 0 {
			return []slashOption{{Cmd: i18n.T(lang, i18n.KeySkillNone), Desc: "", Path: ""}}
		}
		return opts
	}
	// Skill exists: no dropdown; user types natural language after "/skill <name> "
	return []slashOption{}
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
	// Put the manual /remote on option at the end so users see configured remotes first.
	manual := slashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyRemoteManualHint), Path: ""}
	return append(opts, manual)
}

// getRemoveRemoteSlashOptions returns slash options for /config del-remote: one option per configured remote (select to remove).
func getRemoveRemoteSlashOptions(lang string, filter string) []slashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []slashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
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
			Cmd:  "/config del-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
		})
	}
	if len(opts) == 0 {
		return []slashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
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
