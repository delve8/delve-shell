package controller

import (
	"delve-shell/internal/host/bus"
	"delve-shell/internal/hostcmd"
)

func (c *Controller) handleCommand(command hostcmd.Command) {
	switch cmd := command.(type) {
	case hostcmd.Submission:
		userText := cmd.Submission.RawText
		if cmd.Submission.SessionDisplayText != "" {
			userText = cmd.Submission.SessionDisplayText
		}
		c.bus.PublishBlocking(bus.Event{
			Kind:       bus.KindUserChatSubmitted,
			UserText:   userText,
			Submission: cmd.Submission,
		})
	case hostcmd.SessionNew:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionNewRequested})
	case hostcmd.SessionSwitch:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionSwitchRequested, SessionID: cmd.SessionID})
	case hostcmd.ConfigUpdated:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindConfigUpdated})
	case hostcmd.ExecDirect:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindExecDirectRequested, Command: cmd.Command})
	case hostcmd.CancelRequested:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindCancelRequested})
	case hostcmd.ShellSnapshot:
		if c.shellSnapshot != nil {
			snap := hostcmd.ShellSnapshot{
				Messages: append([]string(nil), cmd.Messages...),
				Mode:     cmd.Mode,
			}
			select {
			case c.shellSnapshot <- snap:
			default:
			}
		}
	case hostcmd.RemoteOnTarget:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteOnRequested, RemoteTarget: cmd.Target})
	case hostcmd.RemoteOff:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteOffRequested})
	case hostcmd.RemoteAuthReply:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteAuthResponseSubmitted, RemoteAuthResponse: cmd.Response})
	case hostcmd.AllowlistAutoRun:
		c.currentAllowlistAutoRun.Store(cmd.Enabled)
		if c.runners != nil {
			c.runners.SetAllowlistAutoRun(cmd.Enabled)
		}
	}
}
