package controller

import (
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/route"
	"delve-shell/internal/uivm"
)

func (c *Controller) handleUIAction(action uivm.UIAction) {
	switch action.Kind {
	case uivm.UIActionSubmit:
		c.publishSubmitAction(action.Text)
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
	case uivm.UIActionRequestSlashTrace:
		if action.Text == "" {
			return
		}
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSlashRequested, UserText: action.Text})
	case uivm.UIActionEnterSlashTrace:
		if action.Text == "" {
			return
		}
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSlashEntered, UserText: action.Text})
	}
}

func (c *Controller) publishSubmitAction(text string) {
	classified := route.ClassifyUserSubmit(text)
	switch classified.Kind {
	case route.UserSubmitNewSession:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionNewRequested})
	case route.UserSubmitSwitchSession:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSessionSwitchRequested, SessionID: classified.SessionID})
	default:
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindUserChatSubmitted, UserText: text})
	}
}
