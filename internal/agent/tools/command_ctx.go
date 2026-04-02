package tools

import (
	"context"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/runtime/execcancel"
)

func pushCommandExecutionUI(ch chan<- any) (end func()) {
	if ch == nil {
		return func() {}
	}
	ch <- hiltypes.CommandExecutionState{Active: true}
	return func() {
		ch <- hiltypes.CommandExecutionState{Active: false}
	}
}

func withCommandCancel(h *execcancel.Hub, parent context.Context) (context.Context, func()) {
	if h == nil {
		return parent, func() {}
	}
	return h.WithCancel(parent)
}
