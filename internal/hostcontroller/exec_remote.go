package hostcontroller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/ui"
)

func (c *Controller) handleExecDirect(cmd string) {
	executor := c.getExec()
	stdout, stderrStr, exitCode, runErr := executor.Run(context.Background(), cmd)
	result := stdout
	if stderrStr != "" {
		if result != "" {
			result += "\n"
		}
		result += "stderr:\n" + stderrStr
	}
	result += "\nexit_code: " + fmt.Sprint(exitCode)
	if runErr != nil && exitCode == 0 {
		result += "\nerror: " + runErr.Error()
	}
	c.ui.CommandExecutedDirect(cmd, result)
}

func (c *Controller) handleRemoteOn(target string) {
	identityFile := ""
	label := target
	remotes, errRemotes := config.LoadRemotes()
	if errRemotes == nil && len(remotes) > 0 {
		for _, r := range remotes {
			matched := r.Target == target || r.Name == target || config.HostFromTarget(r.Target) == target
			if matched && r.Target != "" {
				target = r.Target
				identityFile = r.IdentityFile
				hostOnly := config.HostFromTarget(target)
				if r.Name != "" {
					label = fmt.Sprintf("%s (%s)", r.Name, hostOnly)
				} else {
					label = hostOnly
				}
				break
			}
		}
	}
	res := c.executors.Connect(target, label, identityFile)
	c.ui.RemoteAuthPromptPtr(res.AuthPrompt)
	if !res.Connected {
		return
	}
	c.updateRemoteRunCompletion(res.Executor, res.Label)
	c.ui.RemoteStatus(true, res.Label)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
	c.ui.RemoteConnectDone(true, res.Label, "")
}

func (c *Controller) handleRemoteOff() {
	c.executors.SwitchToLocal()
	c.ui.RemoteStatus(false, "")
	c.ui.SystemNotify("Switched back to local executor.")
}

func (c *Controller) handleRemoteAuthResp(resp remoteauth.Response) {
	if resp.Password == "" {
		return
	}
	labelStr, err := c.executors.HandleRemoteAuthResponse(resp)
	if err != nil {
		c.ui.RemoteAuthPrompt(ui.RemoteAuthPromptMsg{
			Target: resp.Target,
			Err:    fmt.Sprintf("Auth failed: %v", err),
		})
		return
	}
	c.updateRemoteRunCompletion(c.getExec(), labelStr)
	c.ui.RemoteStatus(true, labelStr)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", labelStr))
	c.ui.RemoteConnectDone(true, labelStr, "")
}

func (c *Controller) updateRemoteRunCompletion(exec execenv.CommandExecutor, remoteLabel string) {
	if c.currentP.Load() == nil || exec == nil || strings.TrimSpace(remoteLabel) == "" {
		return
	}
	go func() {
		select {
		case <-c.stop:
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
		c.ui.RunCompletionCache(remoteLabel, cmds)
	}()
}
