package hostloop

import (
	"context"
	"sort"
	"strings"
	"time"

	"delve-shell/internal/execenv"
	"delve-shell/internal/ui"
)

// UpdateRemoteRunCompletion fetches a one-time /run completion list from the remote host (best-effort).
func (d *Deps) UpdateRemoteRunCompletion(exec execenv.CommandExecutor, remoteLabel string) {
	if d.CurrentP.Load() == nil || exec == nil || strings.TrimSpace(remoteLabel) == "" {
		return
	}
	go func() {
		select {
		case <-d.Stop:
			return
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		out, _, _, err := exec.Run(ctx, "bash -lc 'compgen -c'")
		if err != nil {
			return
		}
		seen := make(map[string]struct{}, 4096)
		cmds := make([]string, 0, 2048)
		for _, line := range strings.Split(out, "\n") {
			s := strings.TrimSpace(line)
			if s == "" {
				continue
			}
			if strings.ContainsAny(s, " \t/") {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			cmds = append(cmds, s)
			if len(cmds) >= 8000 {
				break
			}
		}
		sort.Strings(cmds)
		d.Send(ui.RunCompletionCacheMsg{RemoteLabel: remoteLabel, Commands: cmds})
	}()
}
