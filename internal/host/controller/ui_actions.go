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
	case uivm.UIActionRelaySlashSubmit:
		if action.SlashSubmit.RawLine == "" {
			return
		}
		p := action.SlashSubmit
		c.bus.PublishBlocking(bus.Event{Kind: bus.KindSlashRelayToUI, SlashSubmit: &p})
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
