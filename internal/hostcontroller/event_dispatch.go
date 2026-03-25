package hostcontroller

import "delve-shell/internal/hostbus"

// hostEventHandlers is the single registration table for bus → controller actions.
// Unregistered kinds are ignored (same as a switch with no default).
var hostEventHandlers = map[hostbus.Kind]func(*Controller, hostbus.Event){
	hostbus.KindSessionNewRequested: func(c *Controller, _ hostbus.Event) {
		c.handleSubmitNewSession()
	},
	hostbus.KindSessionSwitchRequested: func(c *Controller, e hostbus.Event) {
		c.handleSubmitSwitchSession(e.SessionID)
	},
	hostbus.KindUserChatSubmitted: func(c *Controller, e hostbus.Event) {
		c.handleUserChat(e.UserText)
	},
	hostbus.KindConfigUpdated: func(c *Controller, _ hostbus.Event) {
		c.handleConfigUpdated()
	},
	hostbus.KindCancelRequested: func(c *Controller, _ hostbus.Event) {
		c.handleCancelRequest()
	},
	hostbus.KindExecDirectRequested: func(c *Controller, e hostbus.Event) {
		c.handleExecDirect(e.Command)
	},
	hostbus.KindRemoteOnRequested: func(c *Controller, e hostbus.Event) {
		c.handleRemoteOn(e.RemoteTarget)
	},
	hostbus.KindRemoteOffRequested: func(c *Controller, _ hostbus.Event) {
		c.handleRemoteOff()
	},
	hostbus.KindRemoteAuthResponseSubmitted: func(c *Controller, e hostbus.Event) {
		c.handleRemoteAuthResp(e.RemoteAuthResponse)
	},
	hostbus.KindApprovalRequested: func(c *Controller, e hostbus.Event) {
		if e.Approval != nil {
			c.ui.ShowApproval(e.Approval)
		}
	},
	hostbus.KindSensitiveConfirmationRequested: func(c *Controller, e hostbus.Event) {
		if e.Sensitive != nil {
			c.ui.ShowSensitiveConfirmation(e.Sensitive)
		}
	},
	hostbus.KindAgentExecEvent: func(c *Controller, e hostbus.Event) {
		v := e.AgentExec
		c.ui.CommandExecutedFromTool(v.Command, v.Allowed, v.Result, v.Sensitive, v.Suggested)
	},
	hostbus.KindAgentUnknown: func(c *Controller, e hostbus.Event) {
		c.handleAgentUI(e.AgentUI)
	},
	hostbus.KindLLMRunCompleted: func(c *Controller, e hostbus.Event) {
		c.handleLLMRunCompleted(e.Reply, e.Err)
	},
	hostbus.KindSlashRequested: func(c *Controller, e hostbus.Event) {
		c.handleSlashRequested(e)
	},
	hostbus.KindSlashEntered: func(c *Controller, e hostbus.Event) {
		c.handleSlashEntered(e)
	},
}
