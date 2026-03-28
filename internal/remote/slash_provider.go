package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const remoteExecUsageOption = "/exec <cmd>"

func remoteSlashOptionsProvider(
	inputVal string,
	lang string,
) ([]ui.SlashOption, bool) {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)

	if normalizedLower == "remote" || strings.HasPrefix(normalizedLower, "remote ") {
		if normalizedLower == "remote on" {
			hostOpts := getRemoteSlashOptions()
			opts := make([]ui.SlashOption, 0, len(hostOpts)+2)
			opts = append(opts, ui.SlashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyDescRemoteOn)})
			opts = append(opts, hostOpts...)
			opts = append(opts, ui.SlashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff)})
			return opts, true
		}
		return buildRemoteDropdownOptions(lang), true
	}

	if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
		if rest == "del-remote" || strings.HasPrefix(rest, "del-remote ") {
			return getRemoveRemoteSlashOptions(lang), true
		}
	}

	// Remote /exec suggestions: when a cached list exists, prefer it over local PATH scanning.
	if normalizedLower == "exec" || strings.HasPrefix(normalizedLower, "exec ") {
		cands := getCachedRunSuggestions()
		if len(cands) == 0 {
			return nil, false
		}
		if normalizedLower == "exec" {
			return []ui.SlashOption{{Cmd: remoteExecUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun)}}, true
		}
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "exec"))
		if strings.Contains(rest, " ") || strings.Contains(rest, "\t") {
			return []ui.SlashOption{}, true
		}
		prefix := strings.ToLower(rest)
		const maxRunCands = 50
		opts := make([]ui.SlashOption, 0, 12)
		for _, c := range cands {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(c), prefix) {
				continue
			}
			opts = append(opts, ui.SlashOption{Cmd: "/exec " + c, Desc: ""})
			if len(opts) >= maxRunCands {
				break
			}
		}
		return opts, true
	}

	return nil, false
}

func buildRemoteDropdownOptions(lang string) []ui.SlashOption {
	opts := make([]ui.SlashOption, 0, 8)
	seen := make(map[string]struct{}, 8)
	push := func(opt ui.SlashOption) {
		if _, ok := seen[opt.Cmd]; ok {
			return
		}
		seen[opt.Cmd] = struct{}{}
		opts = append(opts, opt)
	}

	for _, opt := range getRemoteSlashOptions() {
		push(opt)
	}
	push(ui.SlashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyDescRemoteOn)})
	push(ui.SlashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff)})
	return opts
}
