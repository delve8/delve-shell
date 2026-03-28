package interactive

import (
	"delve-shell/internal/bootstrap"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/session"
)

// Run starts the interactive TUI loop, host controller, and optional subshell return path.
func Run() error {
	bootstrap.Install()

	stop := make(chan struct{})
	defer close(stop)

	pf, err := RunPreflight()
	if err != nil {
		return err
	}

	sessions := sessionmgr.New(pf.InitialSession)
	syncSessionPath := func(path string) { session.SetCurrentSessionPath(path) }
	syncSessionPath(pf.InitialSession.Path())
	defer sessions.CloseAll()

	stack := wireHostStack(stop, pf, sessions, syncSessionPath)
	loop := newTuiRestartLoop(stack.controller, stack.currentP, stack.shellSnap, stack.commands, pf.NeedConfigLLM, stack.rt, stack.getExec)
	return loop.run()
}
