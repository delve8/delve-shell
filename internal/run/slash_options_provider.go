package run

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func registerSlashOptionsProviders() {
	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)
		if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
			return configSlashOptions(), true
		}
		return nil, false
	})

	ui.RegisterRootSlashOptionProvider(rootSlashOptions)
}

func rootSlashOptions(lang string) []ui.SlashOption {
	opts := []ui.SlashOption{
		{Cmd: "/access", Desc: i18n.T(i18n.KeyDescAccess)},
		{Cmd: "/skill", Desc: i18n.T(i18n.KeyDescSkill)},
	}
	opts = append(opts, bashRootSlashOptions(lang)...)
	opts = append(opts, []ui.SlashOption{
		{Cmd: "/config", Desc: i18n.T(i18n.KeyDescConfig)},
		{Cmd: "/new", Desc: i18n.T(i18n.KeyDescNewSession)},
		{Cmd: "/history", Desc: i18n.T(i18n.KeyDescSessions)},
		{Cmd: "/help", Desc: i18n.T(i18n.KeyDescHelp)},
		{Cmd: "/quit", Desc: i18n.T(i18n.KeyDescExit)},
	}...)
	return opts
}

func configSlashOptions() []ui.SlashOption {
	return []ui.SlashOption{
		{Cmd: "/config remove-remote", Desc: i18n.T(i18n.KeyDescConfigRemoveRemote)},
		{Cmd: "/config model", Desc: i18n.T(i18n.KeyDescConfigModel)},
	}
}
