package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/slash/access"
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

	if normalizedLower == "access" || strings.HasPrefix(normalizedLower, "access ") {
		if normalizedLower == "access" {
			hostOpts := getRemoteSlashOptions(lang)
			opts := make([]ui.SlashOption, 0, len(hostOpts)+2)
			opts = append(opts, hostOpts...)
			opts = append(opts, ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedNew), Desc: i18n.T(i18n.KeyDescAccessNew)})
			opts = append(opts, ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedLocal), Desc: i18n.T(i18n.KeyDescRemoteOff)})
			opts = append(opts, ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedOffline), Desc: i18n.T(i18n.KeyDescAccessOffline)})
			return opts, true
		}
		return buildRemoteDropdownOptions(lang), true
	}

	if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
		if rest == "remove-remote" || strings.HasPrefix(rest, "remove-remote ") {
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
			return []ui.SlashOption{{Cmd: remoteExecUsageOption, Desc: i18n.T(i18n.KeyDescRun)}}, true
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
		key := opt.Cmd + "\x00" + opt.FillValue + "\x00" + opt.ExecuteValue + "\x00" + opt.Desc
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		opts = append(opts, opt)
	}

	for _, opt := range getRemoteSlashOptions(lang) {
		push(opt)
	}
	push(ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedNew), Desc: i18n.T(i18n.KeyDescAccessNew)})
	push(ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedLocal), Desc: i18n.T(i18n.KeyDescRemoteOff)})
	push(ui.SlashOption{Cmd: slashaccess.Command(slashaccess.ReservedOffline), Desc: i18n.T(i18n.KeyDescAccessOffline)})
	return opts
}
