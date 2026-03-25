package controller

import (
	"delve-shell/internal/config"
)

func (c *Controller) handleCancelRequest() {
	if !c.llmRunning || c.llmCancel == nil {
		return
	}
	c.llmCancel()
}

func (c *Controller) handleConfigUpdated() {
	if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
		c.currentAllowlistAutoRun.Store(cfg.AllowlistAutoRunResolved())
	}
	c.runners.SetAllowlistAutoRun(c.currentAllowlistAutoRun.Load())
	c.ui.ConfigReloaded()
}

func (c *Controller) handleAgentUI(x any) {
	c.ui.DispatchAgentUI(x)
}
