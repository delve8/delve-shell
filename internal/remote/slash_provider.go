package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/slash/access"
	"delve-shell/internal/ui"
)

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
