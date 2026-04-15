package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

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
		c.ui.ExecStreamBegin(cmd, false, false, true, false)
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
		c.updateRemoteIssueFromExecError(runErr)
		tail := "exit_code: " + strconv.Itoa(exitCode)
		if runErr != nil && (exitCode == 0 || execenv.IsSSHConnectionError(runErr)) {
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
	c.updateRemoteIssueFromExecError(runErr)
	result := stdout
	if stderrStr != "" {
		if result != "" {
			result += "\n"
		}
		result += "stderr:\n" + stderrStr
	}
	result += "\nexit_code: " + fmt.Sprint(exitCode)
	if runErr != nil && (exitCode == 0 || execenv.IsSSHConnectionError(runErr)) {
		result += "\nerror: " + runErr.Error()
	}
	c.ui.CommandExecutedDirect(cmd, result)
}

func (c *Controller) updateRemoteIssueFromExecError(err error) {
	if c.runtime == nil || !c.runtime.RemoteActive() {
		return
	}
	issue := ""
	var connErr *execenv.SSHConnectionError
	if errors.As(err, &connErr) && !connErr.ReconnectSuccess {
		issue = execenv.SSHConnectionIssueSummary(err)
	}
	c.runtime.SetRemoteIssue(issue)
	c.ui.RemoteStatus(true, c.runtime.RemoteLabel(), false, issue)
}

func (c *Controller) handleAccessRemote(target string, socks5Addr string) {
	if c.runtime != nil {
		c.runtime.SetOffline(false)
	}
	// Drop cached runner: OfflineMode is fixed when the runner is built; without this, Offline → Remote
	// still uses execute_command's offline paste path until something else invalidates.
	if c.runners != nil {
		c.runners.Invalidate()
	}
	resolved := resolveAccessRemoteTarget(target, socks5Addr)
	res := c.executors.Connect(resolved.Target, resolved.Label, resolved.IdentityFile, resolved.Socks5Addr)
	if res.AuthPrompt != nil {
		c.ui.Raw(remote.AuthPromptMsg{
			Target:                res.AuthPrompt.Target,
			Err:                   res.AuthPrompt.Err,
			UseConfiguredIdentity: res.AuthPrompt.UseConfiguredIdentity,
			HostKeyVerify:         res.AuthPrompt.HostKeyVerify,
			HostKeyFingerprint:    res.AuthPrompt.HostKeyFingerprint,
			HostKeyHost:           res.AuthPrompt.HostKeyHost,
			Socks5Addr:            res.AuthPrompt.Socks5Addr,
		})
	}
	if !res.Connected {
		if res.ErrText != "" {
			c.ui.RemoteConnectDone(false, res.Label, res.ErrText)
			c.ui.SystemNotify(res.ErrText)
		}
		return
	}
	if c.runtime != nil {
		c.runtime.SetRemoteExecution(true, res.Label, resolved.HostOnly, resolved.ConfigName)
		c.runtime.SetRemoteIssue("")
	}
	go c.refreshHostMemory(res.Executor, res.Label)
	c.ui.RemoteStatus(true, res.Label, false, "")
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
	c.ui.RemoteConnectDone(true, res.Label, "")
}

type accessRemoteTarget struct {
	Target       string
	Label        string
	HostOnly     string
	ConfigName   string
	IdentityFile string
	Socks5Addr   string
}

func resolveAccessRemoteTarget(input string, socks5Addr string) accessRemoteTarget {
	input = strings.TrimSpace(input)
	socks5Addr = strings.TrimSpace(socks5Addr)
	resolved := accessRemoteTarget{
		Target:     input,
		Label:      input,
		HostOnly:   config.HostFromTarget(input),
		Socks5Addr: socks5Addr,
	}

	if remote, ok := findSavedRemote(input, false); ok {
		remoteSocks5Addr := strings.TrimSpace(remote.Socks5Addr)
		if socks5Addr != "" {
			remoteSocks5Addr = socks5Addr
		}
		return accessRemoteTarget{
			Target:       remote.Target,
			Label:        remoteDisplayLabel(remote),
			HostOnly:     config.HostFromTarget(remote.Target),
			ConfigName:   strings.TrimSpace(remote.Name),
			IdentityFile: strings.TrimSpace(remote.IdentityFile),
			Socks5Addr:   remoteSocks5Addr,
		}
	}
	if sshHost, ok, err := config.ResolveSSHConfigHost(input); err == nil && ok {
		hostOnly := config.HostFromTarget(sshHost.Target)
		label := strings.TrimSpace(sshHost.Alias)
		if label == "" {
			label = hostOnly
		}
		if hostOnly != "" && !strings.EqualFold(label, hostOnly) {
			label = fmt.Sprintf("%s (%s)", label, hostOnly)
		}
		return accessRemoteTarget{
			Target:       sshHost.Target,
			Label:        label,
			HostOnly:     hostOnly,
			ConfigName:   strings.TrimSpace(sshHost.Alias),
			IdentityFile: strings.TrimSpace(sshHost.IdentityFile),
			Socks5Addr:   socks5Addr,
		}
	}
	if remote, ok := findSavedRemote(input, true); ok {
		remoteSocks5Addr := strings.TrimSpace(remote.Socks5Addr)
		if socks5Addr != "" {
			remoteSocks5Addr = socks5Addr
		}
		return accessRemoteTarget{
			Target:       remote.Target,
			Label:        remoteDisplayLabel(remote),
			HostOnly:     config.HostFromTarget(remote.Target),
			ConfigName:   strings.TrimSpace(remote.Name),
			IdentityFile: strings.TrimSpace(remote.IdentityFile),
			Socks5Addr:   remoteSocks5Addr,
		}
	}
	return resolved
}

func findSavedRemote(input string, includeName bool) (config.RemoteTarget, bool) {
	remotes, err := config.LoadRemotes()
	if err != nil {
		return config.RemoteTarget{}, false
	}
	for _, r := range remotes {
		matched := r.Target == input || config.HostFromTarget(r.Target) == input
		if includeName && strings.TrimSpace(r.Name) == input {
			matched = true
		}
		if matched && strings.TrimSpace(r.Target) != "" {
			return r, true
		}
	}
	return config.RemoteTarget{}, false
}

func remoteDisplayLabel(r config.RemoteTarget) string {
	hostOnly := config.HostFromTarget(r.Target)
	if name := strings.TrimSpace(r.Name); name != "" {
		return fmt.Sprintf("%s (%s)", name, hostOnly)
	}
	return hostOnly
}

func (c *Controller) handleAccessLocal() {
	if c.runtime != nil {
		c.runtime.SetOffline(false)
		c.runtime.SetRemoteExecution(false, "", "", "")
		c.runtime.SetRemoteIssue("")
	}
	c.executors.SwitchToLocal()
	if c.runners != nil {
		c.runners.Invalidate()
	}
	c.ui.RemoteStatus(false, "", false, "")
	c.ui.SystemNotify("Switched back to local executor.")
	c.primeHostMemory("local")
}

func (c *Controller) handleAccessOffline() {
	c.executors.SwitchToLocal()
	if c.runtime != nil {
		c.runtime.SetOffline(true)
		c.runtime.SetRemoteIssue("")
	}
	if c.runners != nil {
		c.runners.Invalidate()
	}
	c.ui.RemoteStatus(false, "", true, "")
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
				Socks5Addr:            res.AuthPrompt.Socks5Addr,
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
			c.runtime.SetRemoteIssue("")
		}
		go c.refreshHostMemory(res.Executor, res.Label)
		c.ui.RemoteStatus(true, res.Label, false, "")
		c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
		c.ui.RemoteConnectDone(true, res.Label, "")
		return
	}
	if resp.Password == "" {
		return
	}
	labelStr, err := c.executors.HandleRemoteAuthResponse(resp)
	if err != nil {
		c.ui.Raw(remote.AuthPromptMsg{Target: resp.Target, Socks5Addr: resp.Socks5Addr, Err: fmt.Sprintf("Auth failed: %v", err)})
		return
	}
	if c.runtime != nil {
		c.runtime.SetRemoteExecution(true, labelStr, labelStr, "")
		c.runtime.SetRemoteIssue("")
	}
	go c.refreshHostMemory(c.getExec(), labelStr)
	c.ui.RemoteStatus(true, labelStr, false, "")
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", labelStr))
	c.ui.RemoteConnectDone(true, labelStr, "")
}
