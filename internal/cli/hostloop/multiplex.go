package hostloop

import (
	"context"
	"fmt"

	"delve-shell/internal/config"
	"delve-shell/internal/ui"
)

// RunHostMultiplex multiplexes non-submit host I/O into Send. One select, one goroutine.
func RunHostMultiplex(stop <-chan struct{}, d *Deps) {
	for {
		select {
		case <-stop:
			return
		case path := <-d.SessionSwitchChan:
			_, err := d.Sessions.SwitchTo(path)
			if err != nil {
				d.Send(ui.AgentReplyMsg{Err: err})
				continue
			}
			d.Runners.Invalidate()
			d.Send(ui.SessionSwitchedMsg{Path: path})

		case <-d.ConfigUpdatedChan:
			if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
				d.CurrentAllowlistAutoRun.Store(cfg.AllowlistAutoRunResolved())
			}
			d.Runners.SetAllowlistAutoRun(d.CurrentAllowlistAutoRun.Load())
			d.Send(ui.ConfigReloadedMsg{})

		case newAutoRun := <-d.AllowlistAutoRunChangeChan:
			d.CurrentAllowlistAutoRun.Store(newAutoRun)
			d.Runners.SetAllowlistAutoRun(newAutoRun)

		case x := <-d.UIEvents:
			dispatchAgentUI(d, x)

		case cmd := <-d.ExecDirectChan:
			executor := d.GetExecutor()
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
			d.Send(ui.CommandExecutedMsg{Command: cmd, Direct: true, Result: result})

		case target := <-d.RemoteOnChan:
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
			res := d.Executors.Connect(target, label, identityFile)
			if res.AuthPrompt != nil {
				d.Send(*res.AuthPrompt)
			}
			if !res.Connected {
				continue
			}
			d.UpdateRemoteRunCompletion(res.Executor, res.Label)
			d.Send(ui.RemoteStatusMsg{Active: true, Label: res.Label})
			d.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", res.Label)})
			d.Send(ui.RemoteConnectDoneMsg{Success: true, Label: res.Label})

		case <-d.RemoteOffChan:
			d.Executors.SwitchToLocal()
			d.Send(ui.RemoteStatusMsg{Active: false, Label: ""})
			d.Send(ui.SystemNotifyMsg{Text: "Switched back to local executor."})

		case resp := <-d.RemoteAuthRespChan:
			if resp.Password == "" {
				continue
			}
			labelStr, err := d.Executors.HandleRemoteAuthResponse(resp)
			if err != nil {
				d.Send(ui.RemoteAuthPromptMsg{
					Target: resp.Target,
					Err:    fmt.Sprintf("Auth failed: %v", err),
				})
				continue
			}
			d.UpdateRemoteRunCompletion(d.GetExecutor(), labelStr)
			d.Send(ui.RemoteStatusMsg{Active: true, Label: labelStr})
			d.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", labelStr)})
			d.Send(ui.RemoteConnectDoneMsg{Success: true, Label: labelStr})
		}
	}
}
