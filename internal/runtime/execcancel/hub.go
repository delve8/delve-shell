// Package execcancel coordinates cancelling in-flight shell commands started by tools.
// Multiple registrations may exist concurrently; Esc cancels all of them.
package execcancel

import (
	"context"
	"sync"
)

type registration struct {
	cancel context.CancelFunc
}

// Hub tracks active command cancel funcs (overlapping tool runs may register more than one).
// Cancel invokes every registered cancel still present. Safe for concurrent use.
type Hub struct {
	mu   sync.Mutex
	regs []*registration
}

// New returns a hub; methods are no-ops on nil receiver.
func New() *Hub {
	return &Hub{}
}

// Register stores cancel until unregister runs. Unregister must be called when the command finishes.
func (h *Hub) Register(cancel context.CancelFunc) (unregister func()) {
	if h == nil {
		return func() {}
	}
	r := &registration{cancel: cancel}
	h.mu.Lock()
	h.regs = append(h.regs, r)
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		for i := range h.regs {
			if h.regs[i] == r {
				h.regs = append(h.regs[:i], h.regs[i+1:]...)
				break
			}
		}
		h.mu.Unlock()
	}
}

// WithCancel returns a child of parent and registers its Cancel for ESC handling.
func (h *Hub) WithCancel(parent context.Context) (context.Context, func()) {
	if h == nil {
		return parent, func() {}
	}
	ctx, cancel := context.WithCancel(parent)
	unreg := h.Register(cancel)
	return ctx, func() {
		unreg()
	}
}

// Cancel invokes every active command's cancel func. Returns whether at least one registration existed.
func (h *Hub) Cancel() bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	n := len(h.regs)
	var toCall []context.CancelFunc
	if n > 0 {
		toCall = make([]context.CancelFunc, 0, n)
		for _, r := range h.regs {
			toCall = append(toCall, r.cancel)
		}
	}
	h.mu.Unlock()
	if len(toCall) == 0 {
		return false
	}
	for _, c := range toCall {
		c()
	}
	return true
}
