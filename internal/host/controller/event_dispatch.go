package controller

import "delve-shell/internal/host/bus"

// hostEventHandlers is the single registration table for bus → controller actions.
// Unregistered kinds are ignored (same as a switch with no default).
var hostEventHandlers = map[bus.Kind]func(*Controller, bus.Event){
	bus.KindSessionNewRequested: func(c *Controller, _ bus.Event) {
		c.handleSubmitNewSession()
	},
	bus.KindSessionSwitchRequested: func(c *Controller, e bus.Event) {
		c.handleSubmitSwitchSession(e.SessionID)
	},
	bus.KindUserChatSubmitted: func(c *Controller, e bus.Event) {
		c.handleUserChat(e)
	},
	bus.KindConfigUpdated: func(c *Controller, _ bus.Event) {
		c.handleConfigUpdated()
	},
	bus.KindCancelRequested: func(c *Controller, _ bus.Event) {
		c.handleCancelRequest()
	},
	bus.KindExecDirectRequested: func(c *Controller, e bus.Event) {
		c.handleExecDirect(e.Command)
	},
	bus.KindRemoteOnRequested: func(c *Controller, e bus.Event) {
		c.handleRemoteOn(e.RemoteTarget)
	},
	bus.KindRemoteOffRequested: func(c *Controller, _ bus.Event) {
		c.handleRemoteOff()
	},
	bus.KindRemoteAuthResponseSubmitted: func(c *Controller, e bus.Event) {
		c.handleRemoteAuthResp(e.RemoteAuthResponse)
	},
	bus.KindApprovalRequested: func(c *Controller, e bus.Event) {
		if e.Approval != nil {
			c.ui.ShowApproval(e.Approval)
		}
	},
	bus.KindSensitiveConfirmationRequested: func(c *Controller, e bus.Event) {
		if e.Sensitive != nil {
			c.ui.ShowSensitiveConfirmation(e.Sensitive)
		}
	},
	bus.KindAgentExecEvent: func(c *Controller, e bus.Event) {
		v := e.AgentExec
		c.ui.CommandExecutedFromTool(v.Command, v.Allowed, v.Result, v.Sensitive, v.Suggested)
	},
	bus.KindAgentUnknown: func(c *Controller, e bus.Event) {
		c.handleAgentUI(e.AgentUI)
	},
	bus.KindLLMRunCompleted: func(c *Controller, e bus.Event) {
		c.handleLLMRunCompleted(e.Reply, e.Err)
	},
}
