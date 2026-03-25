package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func remoteSlashOptionsProvider(
	inputVal string,
	lang string,
	_ []string,
	_ bool,
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

	return nil, false
}
