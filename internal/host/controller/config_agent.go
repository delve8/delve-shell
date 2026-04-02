package controller

func (c *Controller) handleCancelRequest() {
	if c.execCancelHub != nil && c.execCancelHub.Cancel() {
		// Immediate transcript line; do not wait until Run/RunStreaming returns (slow if the child ignores signals).
		c.ui.SystemNotify("Execution cancelled.")
		return
	}
	if !c.llmRunning || c.llmCancel == nil {
		return
	}
	c.llmCancel()
}

func (c *Controller) handleConfigUpdated() {
	if c.runners != nil {
		c.runners.Invalidate()
	}
	c.ui.ConfigReloaded()
}

func (c *Controller) handleAgentUI(x any) {
	c.ui.DispatchAgentUI(x)
}
