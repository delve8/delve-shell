package controller

import (
	"delve-shell/internal/host/bus"
	"delve-shell/internal/uivm"
)

func (c *Controller) handleUIAction(action uivm.UIAction) {
	switch action.Kind {
	case uivm.UIActionSubmission:
		c.bus.PublishBlocking(bus.Event{
			Kind:       bus.KindUserChatSubmitted,
			UserText:   action.Submission.RawText,
			Submission: action.Submission,
		})
	case uivm.UIActionSessionNew:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionNewRequested})
	case uivm.UIActionSessionSwitch:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionSwitchRequested, SessionID: action.Text})
	case uivm.UIActionConfigUpdated:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindConfigUpdated})
	case uivm.UIActionExecDirect:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindExecDirectRequested, Command: action.Text})
	case uivm.UIActionCancelRequested:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindCancelRequested})
	case uivm.UIActionShellSnapshot:
		if c.shellSnapshot != nil {
			msgs := make([]string, len(action.Messages))
			copy(msgs, action.Messages)
			select {
			case c.shellSnapshot <- msgs:
			default:
			}
		}
	case uivm.UIActionRemoteOnTarget:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteOnRequested, RemoteTarget: action.Text})
	case uivm.UIActionRemoteOff:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteOffRequested})
	case uivm.UIActionRemoteAuthReply:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindRemoteAuthResponseSubmitted, RemoteAuthResponse: action.RemoteAuthReply})
	case uivm.UIActionAllowlistAutoRun:
		c.currentAllowlistAutoRun.Store(action.BoolValue)
		if c.runners != nil {
			c.runners.SetAllowlistAutoRun(action.BoolValue)
		}
	}
}
