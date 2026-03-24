package run

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashOptionsProvider(func(
		inputVal string,
		lang string,
		_ string,
		localRunCommands []string,
		remoteRunCommands []string,
		remoteActive bool,
	) ([]ui.SlashOption, bool) {
		normalized := strings.TrimPrefix(inputVal, "/")
		normalized = strings.TrimSpace(normalized)
		normalizedLower := strings.ToLower(normalized)
		if normalizedLower == "config" || strings.HasPrefix(normalizedLower, "config ") {
			return configSlashOptions(lang), true
		}
		if normalizedLower != "run" && !strings.HasPrefix(normalizedLower, "run ") {
			return nil, false
		}
		if normalizedLower == "run" {
			return []ui.SlashOption{{Cmd: ui.SlashRunUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun), Path: ""}}, true
		}
		rest := ""
		if len(normalized) >= 3 {
			rest = strings.TrimSpace(normalized[3:])
		}
		if strings.Contains(rest, " ") || strings.Contains(rest, "\t") {
			return []ui.SlashOption{}, true
		}
		prefix := strings.ToLower(rest)
		cands := localRunCommands
		if cands == nil {
			cands = loadLocalRunCommands()
		}
		if remoteActive && len(remoteRunCommands) > 0 {
			cands = remoteRunCommands
		}
		const maxRunCands = 50
		opts := make([]ui.SlashOption, 0, 12)
		for _, c := range cands {
			if prefix != "" && !strings.HasPrefix(strings.ToLower(c), prefix) {
				continue
			}
			opts = append(opts, ui.SlashOption{Cmd: "/run " + c, Desc: "", Path: ""})
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
	return []ui.SlashOption{
		{Cmd: "/help", Desc: i18n.T(lang, i18n.KeyDescHelp), Path: ""},
		{Cmd: "/cancel", Desc: i18n.T(lang, i18n.KeyDescCancel), Path: ""},
		{Cmd: "/config", Desc: i18n.T(lang, i18n.KeyDescConfig), Path: ""},
		{Cmd: "/remote", Desc: i18n.T(lang, i18n.KeyDescRemoteOn), Path: ""},
		{Cmd: "/new", Desc: i18n.T(lang, i18n.KeySessionNew), Path: ""},
		{Cmd: "/sessions", Desc: i18n.T(lang, i18n.KeyDescSessions), Path: ""},
		{Cmd: "/skill <skill-name> [detail]", Desc: i18n.T(lang, i18n.KeyDescSkill), Path: ""},
		{Cmd: ui.SlashRunUsageOption, Desc: i18n.T(lang, i18n.KeyDescRun), Path: ""},
		{Cmd: "/sh", Desc: i18n.T(lang, i18n.KeyDescSh), Path: ""},
		{Cmd: "/q", Desc: i18n.T(lang, i18n.KeyDescExit), Path: ""},
	}
}

func configSlashOptions(lang string) []ui.SlashOption {
	return []ui.SlashOption{
		{Cmd: "/config add-remote", Desc: i18n.T(lang, i18n.KeyDescConfigAddRemote), Path: ""},
		{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyDescConfigRemoveRemote), Path: ""},
		{Cmd: "/config add-skill", Desc: i18n.T(lang, i18n.KeyDescSkillInstall), Path: ""},
		{Cmd: "/config del-skill", Desc: i18n.T(lang, i18n.KeyDescSkillRemove), Path: ""},
		{Cmd: "/config update-skill", Desc: i18n.T(lang, i18n.KeyDescConfigUpdateSkill), Path: ""},
		{Cmd: "/config auto-run list-only", Desc: i18n.T(lang, i18n.KeyDescAutoRunListOnly), Path: ""},
		{Cmd: "/config auto-run disable", Desc: i18n.T(lang, i18n.KeyDescAutoRunDisable), Path: ""},
		{Cmd: "/config update auto-run list", Desc: i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate), Path: ""},
		{Cmd: "/config llm", Desc: i18n.T(lang, i18n.KeyDescConfigLLM), Path: ""},
		{Cmd: "/config reload", Desc: i18n.T(lang, i18n.KeyDescReload), Path: ""},
	}
}
