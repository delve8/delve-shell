package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const remoteRunUsageOption = "/run <cmd>"

func remoteSlashOptionsProvider(
	inputVal string,
	lang string,
) ([]ui.SlashOption, bool) {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)

	if normalizedLower == "remote" {
		return []ui.SlashOption{
			{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyDescRemoteOn)},
			{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff)},
		}, true
	}

	if strings.HasPrefix(normalizedLower, "remote on") {
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "remote on"))
		opts := getRemoteSlashOptions(filter, lang)
		offOpt := ui.SlashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff)}
		return append([]ui.SlashOption{offOpt}, opts...), true
	}

	// /remote off (and prefixes like "remote o" while typing); do not match "remote on" above.
	if normalizedLower == "remote off" || strings.HasPrefix(normalizedLower, "remote off ") ||
		(strings.HasPrefix(normalizedLower, "remote") && strings.HasPrefix("remote off", normalizedLower) &&
			!strings.HasPrefix(normalizedLower, "remote on")) {
		return []ui.SlashOption{
			{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff)},
		}, true
	}

	if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "config"))
		if rest == "del-remote" || strings.HasPrefix(rest, "del-remote ") {
			filter := strings.TrimSpace(strings.TrimPrefix(rest, "del-remote"))
			return getRemoveRemoteSlashOptions(lang, filter), true
		}
	}

	// Remote /run suggestions: when a cached list exists, prefer it over local PATH scanning.
	if normalizedLower == "run" || strings.HasPrefix(normalizedLower, "run ") {
		cands := getCachedRunSuggestions()
		if len(cands) == 0 {
			return nil, false
		}
		if normalizedLower == "run" {
			return []ui.SlashOption{{Cmd: remoteRunUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun)}}, true
		}
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "run"))
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
			opts = append(opts, ui.SlashOption{Cmd: "/run " + c, Desc: ""})
			if len(opts) >= maxRunCands {
				break
			}
		}
		return opts, true
	}

	return nil, false
}
