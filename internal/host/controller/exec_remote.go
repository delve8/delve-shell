package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/remote"
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/uivm"
)

func (c *Controller) handleExecDirect(cmd string) {
	if c.runtime != nil && c.runtime.Offline() {
		c.ui.TranscriptAppend([]uivm.Line{
			{Kind: uivm.LineSystemError, Text: "Direct execution is disabled in Offline mode. Use the assistant's execute_command flow and paste results back."},
			{Kind: uivm.LineBlank},
		})
		return
	}
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
	if c.runtime != nil {
		c.runtime.SetOffline(false)
	}
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
	if res.AuthPrompt != nil {
		c.ui.Raw(remote.AuthPromptMsg{
			Target:                res.AuthPrompt.Target,
			Err:                   res.AuthPrompt.Err,
			UseConfiguredIdentity: res.AuthPrompt.UseConfiguredIdentity,
			HostKeyVerify:         res.AuthPrompt.HostKeyVerify,
			HostKeyFingerprint:    res.AuthPrompt.HostKeyFingerprint,
			HostKeyHost:           res.AuthPrompt.HostKeyHost,
		})
	}
	if !res.Connected {
		return
	}
	c.updateRemoteRunCompletion(res.Executor, res.Label)
	c.ui.RemoteStatus(true, res.Label, false)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
	c.ui.RemoteConnectDone(true, res.Label, "")
}

func (c *Controller) handleRemoteOff() {
	if c.runtime != nil {
		c.runtime.SetOffline(false)
	}
	c.executors.SwitchToLocal()
	if c.runners != nil {
		c.runners.Invalidate()
	}
	c.ui.RemoteStatus(false, "", false)
	c.ui.SystemNotify("Switched back to local executor.")
}

func (c *Controller) handleAccessOffline() {
	c.executors.SwitchToLocal()
	if c.runtime != nil {
		c.runtime.SetOffline(true)
	}
	if c.runners != nil {
		c.runners.Invalidate()
	}
	c.ui.RemoteStatus(false, "", true)
	c.ui.SystemNotify("Offline mode: commands are not executed here—copy to your environment, paste output in the dialog. Review each command; allowlist is not used.")
}

func (c *Controller) handleRemoteAuthResp(resp remoteauth.Response) {
	if resp.Kind == "hostkey_accept" || resp.Kind == "hostkey_reject" {
		res := c.executors.ResolveHostKeyDecision(resp.Target, resp.Kind == "hostkey_accept")
		if res.AuthPrompt != nil {
			c.ui.Raw(remote.AuthPromptMsg{
				Target:                res.AuthPrompt.Target,
				Err:                   res.AuthPrompt.Err,
				UseConfiguredIdentity: res.AuthPrompt.UseConfiguredIdentity,
				HostKeyVerify:         res.AuthPrompt.HostKeyVerify,
				HostKeyFingerprint:    res.AuthPrompt.HostKeyFingerprint,
				HostKeyHost:           res.AuthPrompt.HostKeyHost,
			})
		}
		if !res.Connected {
			if resp.Kind == "hostkey_reject" {
				label := strings.TrimSpace(res.Label)
				if label == "" {
					label = config.HostFromTarget(resp.Target)
				}
				c.ui.SystemNotify(fmt.Sprintf("Remote host key rejected; not connected to %s.", label))
			}
			c.ui.RemoteConnectDone(false, res.Label, "")
			return
		}
		c.updateRemoteRunCompletion(res.Executor, res.Label)
		c.ui.RemoteStatus(true, res.Label, false)
		c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
		c.ui.RemoteConnectDone(true, res.Label, "")
		return
	}
	if resp.Password == "" {
		return
	}
	labelStr, err := c.executors.HandleRemoteAuthResponse(resp)
	if err != nil {
		c.ui.Raw(remote.AuthPromptMsg{Target: resp.Target, Err: fmt.Sprintf("Auth failed: %v", err)})
		return
	}
	c.updateRemoteRunCompletion(c.getExec(), labelStr)
	c.ui.RemoteStatus(true, labelStr, false)
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
