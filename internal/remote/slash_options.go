package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func getRemoteSlashOptions(filter string, lang string) []ui.SlashOption {
	manualOpt := ui.SlashOption{Cmd: "/remote on", Desc: i18n.T(lang, i18n.KeyRemoteManualHint)}
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []ui.SlashOption{manualOpt}
	}

	filterLower := strings.ToLower(filter)
	var hostOpts []ui.SlashOption
	for _, r := range remotes {
		line := r.Target + " " + r.Name
		if filterLower != "" && !strings.Contains(strings.ToLower(line), filterLower) {
			continue
		}
		desc := r.Name
		hostOpts = append(hostOpts, ui.SlashOption{
			Cmd:  "/remote on " + config.HostFromTarget(r.Target),
			Desc: desc,
		})
	}
	if len(hostOpts) == 0 {
		return []ui.SlashOption{manualOpt}
	}
	return append([]ui.SlashOption{manualOpt}, hostOpts...)
}

func getRemoveRemoteSlashOptions(lang string, filter string) []ui.SlashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone)}}
	}

	manualBase := ui.SlashOption{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyDelRemoteManualHint)}

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
		})
	}

	if len(opts) == 0 {
		return []ui.SlashOption{{Cmd: "/config del-remote", Desc: i18n.T(lang, i18n.KeyRemoteNone)}}
	}
	return append([]ui.SlashOption{manualBase}, opts...)
}
