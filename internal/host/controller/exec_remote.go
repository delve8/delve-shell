package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/host/app"
	"delve-shell/internal/remote"
	remoteauth "delve-shell/internal/remote/auth"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/ui/uivm"
)

func (c *Controller) handleExecDirect(cmd string) {
	if c.runtime != nil && c.runtime.Offline() {
		c.ui.TranscriptAppend([]uivm.Line{
			{Kind: uivm.LineSystemError, Text: "Direct execution is disabled in Offline mode. Use the assistant's execute_command flow and paste results back."},
			{Kind: uivm.LineBlank},
		})
		return
	}
	go func() {
		ctx := context.Background()
		unreg := func() {}
		if c.execCancelHub != nil {
			ctx, unreg = c.execCancelHub.WithCancel(context.Background())
		}
		defer unreg()
		defer c.ui.CommandExecutionActive(false)
		// Register cancel before [EXECUTING] so Esc cannot arrive before hub.Cancel is wired.
		c.ui.CommandExecutionActive(true)
		c.runDirectExecWithContext(ctx, cmd)
	}()
}

func (c *Controller) runDirectExecWithContext(ctx context.Context, cmd string) {
	executor := c.getExec()
	if sr, ok := executor.(execenv.StreamingRunner); ok {
		c.ui.ExecStreamBegin(cmd, false, false, true)
		var outBuf, errBuf bytes.Buffer
		lineOut := execenv.NewLineEmitWriter(func(line string) {
			c.ui.ExecStreamLineOut(line, false)
		})
		lineErr := execenv.NewLineEmitWriter(func(line string) {
			c.ui.ExecStreamLineOut(line, true)
		})
		mwOut := io.MultiWriter(&outBuf, lineOut)
		mwErr := io.MultiWriter(&errBuf, lineErr)
		exitCode, runErr := sr.RunStreaming(ctx, cmd, mwOut, mwErr)
		lineOut.Flush()
		lineErr.Flush()
		cancelled := errors.Is(ctx.Err(), context.Canceled) || errors.Is(runErr, context.Canceled)
		if cancelled {
			// "Execution cancelled." is sent when Esc is handled ([handleCancelRequest]); only close the streamed block here.
			c.ui.CommandExecutedStreamEnd(false, "")
			return
		}
		tail := "exit_code: " + strconv.Itoa(exitCode)
		if runErr != nil && exitCode == 0 {
			tail += "\nerror: " + runErr.Error()
		}
		c.ui.CommandExecutedStreamEnd(false, tail)
		return
	}

	stdout, stderrStr, exitCode, runErr := executor.Run(ctx, cmd)
	cancelled := errors.Is(ctx.Err(), context.Canceled) || errors.Is(runErr, context.Canceled)
	if cancelled {
		c.ui.CommandExecutedDirect(cmd, "")
		return
	}
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

func (c *Controller) handleAccessRemote(target string) {
	if c.runtime != nil {
		c.runtime.SetOffline(false)
	}
	// Drop cached runner: OfflineMode is fixed when the runner is built; without this, Offline → Remote
	// still uses execute_command's offline paste path until something else invalidates.
	if c.runners != nil {
		c.runners.Invalidate()
	}
	identityFile := ""
	label := target
	hostOnly := config.HostFromTarget(target)
	cfgName := ""
	remotes, errRemotes := config.LoadRemotes()
	if errRemotes == nil && len(remotes) > 0 {
		for _, r := range remotes {
			matched := r.Target == target || r.Name == target || config.HostFromTarget(r.Target) == target
			if matched && r.Target != "" {
				target = r.Target
				identityFile = r.IdentityFile
				hostOnly = config.HostFromTarget(target)
				cfgName = strings.TrimSpace(r.Name)
				if cfgName != "" {
					label = fmt.Sprintf("%s (%s)", cfgName, hostOnly)
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
	if c.runtime != nil {
		c.runtime.SetRemoteExecution(true, res.Label, hostOnly, cfgName)
	}
	c.updateRemoteRunCompletion(res.Executor, res.Label)
	c.ui.RemoteStatus(true, res.Label, false)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
	c.ui.RemoteConnectDone(true, res.Label, "")
}

func (c *Controller) handleAccessLocal() {
	if c.runtime != nil {
		c.runtime.SetOffline(false)
		c.runtime.SetRemoteExecution(false, "", "", "")
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
	c.ui.SystemNotify("Offline mode: commands are shown only, not executed here. Paste the results back and review them before running them elsewhere.")
}

func (c *Controller) handleRemoteAuthResp(resp remoteauth.Response) {
	if resp.Kind == remoteauth.ResponseKindHostKeyAccept || resp.Kind == remoteauth.ResponseKindHostKeyReject {
		res := c.executors.ResolveHostKeyDecision(resp.Target, resp.Kind == remoteauth.ResponseKindHostKeyAccept)
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
			if resp.Kind == remoteauth.ResponseKindHostKeyReject {
				label := strings.TrimSpace(res.Label)
				if label == "" {
					label = config.HostFromTarget(resp.Target)
				}
				c.ui.SystemNotify(fmt.Sprintf("Remote host key rejected; not connected to %s.", label))
			}
			c.ui.RemoteConnectDone(false, res.Label, "")
			return
		}
		if c.runtime != nil {
			n, h := app.ParseRemoteDisplayLabel(res.Label)
			if h == "" {
				h = config.HostFromTarget(resp.Target)
			}
			c.runtime.SetRemoteExecution(true, res.Label, h, n)
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
	if c.runtime != nil {
		c.runtime.SetRemoteExecution(true, labelStr, labelStr, "")
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
