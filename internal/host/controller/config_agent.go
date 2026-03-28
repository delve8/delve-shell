package controller

func (c *Controller) handleCancelRequest() {
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
