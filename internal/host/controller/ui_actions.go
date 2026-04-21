package controller

import (
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/cmd"
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
	case hostcmd.HistoryPreviewOpen:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindHistoryPreviewRequested, SessionID: cmd.SessionID})
	case hostcmd.ConfigUpdated:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindConfigUpdated})
	case hostcmd.CancelRequested:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindCancelRequested})
	case hostcmd.ShellSnapshot:
		if c.shellSnapshot != nil {
			snap := hostcmd.ShellSnapshot{
				Messages:     append([]string(nil), cmd.Messages...),
				InputHistory: append([]string(nil), cmd.InputHistory...),
				Mode:         cmd.Mode,
			}
			select {
			case c.shellSnapshot <- snap:
			default:
			}
		}
	case hostcmd.AccessRemote:
		c.bus.PublishBlocking(bus.Event{
			Kind:             bus.KindAccessRemoteRequested,
			RemoteTarget:     cmd.Target,
			RemoteSocks5Addr: cmd.Socks5Addr,
		})
	case hostcmd.AccessLocal:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindAccessLocalRequested})
	case hostcmd.AccessOffline:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindAccessOfflineRequested})
	case hostcmd.RemoteAuthReply:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteAuthResponseSubmitted, RemoteAuthResponse: cmd.Response})
	}
}
