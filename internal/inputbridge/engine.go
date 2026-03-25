package inputbridge

import (
	"delve-shell/internal/inputlifecycle"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputpreflight"
	"delve-shell/internal/inputprocess/chatproc"
	"delve-shell/internal/inputprocess/controlproc"
	"delve-shell/internal/inputprocess/slashproc"
)

// NewEngine wires the standard migration-time lifecycle engine.
func NewEngine(sink ActionSink, contexts controlproc.ContextProvider, slashExecutor slashproc.Executor) inputlifecycle.Engine {
	router := inputlifecycle.NewRouter(
		controlproc.New(contexts, ControlActionExecutor{Sink: sink}),
		slashproc.New(slashExecutor),
		chatproc.New(ChatActionExecutor{Sink: sink}),
	)
	return inputlifecycle.NewEngine(inputpreflight.Engine{}, router)
}

var _ controlproc.ContextProvider = controlContextProvider{}

type controlContextProvider struct {
	ctx inputlifecycletype.ControlContext
}

func (c controlContextProvider) ControlContext() inputlifecycletype.ControlContext { return c.ctx }
