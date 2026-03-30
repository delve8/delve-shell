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
			return configSlashOptions(lang), true
		}
		if normalizedLower != "exec" && !strings.HasPrefix(normalizedLower, "exec ") {
			return nil, false
		}
		if normalizedLower == "exec" {
			return []ui.SlashOption{{Cmd: slashExecUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun)}}, true
		}
		rest := strings.TrimSpace(strings.TrimPrefix(normalizedLower, "exec"))
		if strings.Contains(rest, " ") || strings.Contains(rest, "\t") {
			return []ui.SlashOption{}, true
		}
		prefix := strings.ToLower(rest)
		cands := loadLocalRunCommands()
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
	})

	ui.RegisterRootSlashOptionProvider(func(lang string) []ui.SlashOption {
		return rootSlashOptions(lang)
	})
}

func rootSlashOptions(lang string) []ui.SlashOption {
	opts := []ui.SlashOption{
		{Cmd: "/access", Desc: i18n.T(lang, i18n.KeyDescRemoteOn)},
		{Cmd: "/skill <name> [detail]", Desc: i18n.T(lang, i18n.KeyDescSkill)},
		{Cmd: slashExecUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun)},
	}
	opts = append(opts, bashRootSlashOptions(lang)...)
	opts = append(opts, []ui.SlashOption{
		{Cmd: "/config", Desc: i18n.T(lang, i18n.KeyDescConfig)},
		{Cmd: "/new", Desc: i18n.T(lang, i18n.KeySessionNew)},
		{Cmd: "/history", Desc: i18n.T(lang, i18n.KeyDescSessions)},
		{Cmd: "/help", Desc: i18n.T(lang, i18n.KeyDescHelp)},
		{Cmd: "/quit", Desc: i18n.T(lang, i18n.KeyDescExit)},
	}...)
	return opts
}

func configSlashOptions(lang string) []ui.SlashOption {
	return []ui.SlashOption{
		{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyDescConfigRemoveRemote)},
		{Cmd: "/config add-skill", Desc: i18n.T(lang, i18n.KeyDescSkillInstall)},
		{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeyDescSkillRemove)},
		{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeyDescConfigUpdateSkill)},
		{Cmd: "/config update auto-run list", Desc: i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate)},
		{Cmd: "/config model", Desc: i18n.T(lang, i18n.KeyDescConfigLLM)},
	}
}
