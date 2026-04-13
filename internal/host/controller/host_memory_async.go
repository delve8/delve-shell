package controller

import "delve-shell/internal/history"

func (c *Controller) bindCurrentSessionHooks() {
	if c == nil || c.sessions == nil {
		return
	}
	s := c.sessions.Current()
	if s == nil {
		return
	}
	s.SetAfterAppendHook(func(ev history.Event) {
		c.handleSessionAppended(s, ev)
	})
}

func (c *Controller) handleSessionAppended(s *history.Session, ev history.Event) {
	if c == nil || s == nil || c.hostMemoryUpdater == nil || c.runtime == nil || c.runtime.Offline() {
		return
	}
	c.hostMemoryUpdater.Enqueue(s.Path(), c.runtime.HostMemoryContext(), ev)
}
