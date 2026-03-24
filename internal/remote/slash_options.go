package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func getRemoteSlashOptions(filter string, lang string) []ui.SlashOption {
	var opts []ui.SlashOption
	remotes, err := config.LoadRemotes()
	if err == nil && len(remotes) > 0 {
		filterLower := strings.ToLower(filter)
		for _, r := range remotes {
			line := r.Target + " " + r.Name
			if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
				continue
			}
			desc := r.Name
			opts = append(opts, ui.SlashOption{
				Cmd:  "/remote on " + config.HostFromTarget(r.Target),
				Desc: desc,
				Path: "",
			})
		}
	}

	manual := ui.SlashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyRemoteManualHint), Path: ""}
	return append(opts, manual)
}

func getRemoveRemoteSlashOptions(lang string, filter string) []ui.SlashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}

	filterLower := strings.ToLower(filter)
	var opts []ui.SlashOption
	for _, r := range remotes {
		line := r.Target + " " + r.Name
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}

		desc := r.Name
		opts = append(opts, ui.SlashOption{
			Cmd:  "/config del-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
			Path: "",
		})
	}

	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone), Path: ""}}
	}
	return opts
}
