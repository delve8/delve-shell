package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func remoteSlashOptionsProvider(
	inputVal string,
	lang string,
	_ string,
	_ []string,
	_ []string,
	_ bool,
) ([]ui.SlashOption, bool) {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.TrimSpace(normalized)
	normalizedLower := strings.ToLower(normalized)

	if normalizedLower == "remote" {
		return []ui.SlashOption{
			{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyDescRemoteOn), Path: ""},
			{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff), Path: ""},
		}, true
	}

	if strings.HasPrefix(normalizedLower, "remote on") {
		filter := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "remote on"))
		opts := getRemoteSlashOptions(filter, lang)
		offOpt := ui.SlashOption{Cmd: "/remote off", Desc: i18n.T(lang, i18n.KeyDescRemoteOff), Path: ""}
		return append([]ui.SlashOption{offOpt}, opts...), true
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
