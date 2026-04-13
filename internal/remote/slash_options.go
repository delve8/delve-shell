package remote

import (
	"fmt"
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// getRemoteSlashOptions returns one row per configured remote. Filtering by what the user typed
// is done by [slashview.VisibleIndices] (same prefix rules as other slash commands, using full Cmd).
func getRemoteSlashOptions(lang string) []ui.SlashOption {
	seen := make(map[string]struct{}, 8)
	hostOpts := make([]ui.SlashOption, 0, 8)

	remotes, err := config.LoadRemotes()
	if err == nil {
		for _, r := range remotes {
			suffix := strings.ToLower(config.HostFromTarget(r.Target))
			if suffix == "" {
				continue
			}
			if _, ok := seen[suffix]; ok {
				continue
			}
			seen[suffix] = struct{}{}
			hostOpts = append(hostOpts, ui.SlashOption{
				Cmd:  "/access " + suffix,
				Desc: r.Name,
			})
		}
	}

	sshHosts, err := config.LoadSSHConfigHosts()
	if err == nil {
		for _, h := range sshHosts {
			displaySuffix := strings.ToLower(strings.TrimSpace(h.HostName))
			if displaySuffix == "" {
				displaySuffix = strings.ToLower(strings.TrimSpace(h.Alias))
			}
			fillSuffix := strings.ToLower(strings.TrimSpace(h.Alias))
			if displaySuffix == "" || fillSuffix == "" {
				continue
			}
			if _, ok := seen[displaySuffix]; ok {
				continue
			}
			seen[displaySuffix] = struct{}{}
			desc := i18n.T(i18n.KeyDescAccessSSHConfig)
			if alias := strings.TrimSpace(h.Alias); alias != "" {
				desc = fmt.Sprintf("%s (from %s)", alias, desc)
			}
			hostOpts = append(hostOpts, ui.SlashOption{
				Cmd:       "/access " + displaySuffix,
				Desc:      desc,
				FillValue: "/access " + displaySuffix,
			})
		}
	}
	return hostOpts
}

// getRemoveRemoteSlashOptions returns one row per configured remote for /config remove-remote.
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
			Cmd:  "/config remove-remote " + config.HostFromTarget(r.Target),
			Desc: desc,
		})
	}
	return opts
}
