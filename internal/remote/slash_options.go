package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// getRemoteSlashOptions returns one row per configured remote. Filtering by what the user typed
// is done by [slashview.VisibleIndices] (same prefix rules as other slash commands, using full Cmd).
func getRemoteSlashOptions() []ui.SlashOption {
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return nil
	}

	var hostOpts []ui.SlashOption
	for _, r := range remotes {
		desc := r.Name
		hostOpts = append(hostOpts, ui.SlashOption{
			Cmd:  "/access " + strings.ToLower(config.HostFromTarget(r.Target)),
			Desc: desc,
		})
	}
	if len(hostOpts) == 0 {
		return nil
	}
	return hostOpts
}

// getRemoveRemoteSlashOptions returns one row per configured remote for /config del-remote.
// Prefix filtering uses [slashview.VisibleIndices] like other slash rows.
func getRemoveRemoteSlashOptions(lang string) []ui.SlashOption {
	noneRow := ui.SlashOption{Cmd: i18n.T(i18n.KeyDelRemoteNoHosts), Desc: ""}
	remotes, err := config.LoadRemotes()
	if err != nil || len(remotes) == 0 {
		return []ui.SlashOption{noneRow}
	}

	var opts []ui.SlashOption
	for _, r := range remotes {
		desc := r.Name
		opts = append(opts, ui.SlashOption{
			Cmd:  "/config del-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
		})
	}
	return opts
}
