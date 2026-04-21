package controller

import (
	"context"
	"time"

	"delve-shell/internal/hostmem"
)

func (c *Controller) primeHostMemory(alias string) {
	if c == nil || c.runtime == nil || c.runtime.Offline() || c.getExec == nil {
		return
	}
	exec := c.getExec()
	if exec == nil {
		return
	}
	go c.refreshHostMemory(exec, alias)
}

func (c *Controller) refreshHostMemory(executor interface {
	Run(context.Context, string) (string, string, int, error)
}, alias string) {
	if c == nil || c.runtime == nil || c.runtime.Offline() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	probe, err := hostmem.Probe(ctx, executor, alias)
	if err != nil {
		return
	}
	memCtx, err := hostmem.ApplyProbe(probe)
	if err != nil {
		return
	}
	c.runtime.SetHostMemoryContext(memCtx)
}
