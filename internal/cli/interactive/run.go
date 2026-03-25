package interactive

import (
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/session"

	// Side-effect: register slash handlers and overlay wiring for the TUI.
	_ "delve-shell/internal/remote"
	_ "delve-shell/internal/run"
)

// Run starts the interactive TUI loop, host controller, and optional subshell return path.
func Run() error {
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
	loop := newTuiRestartLoop(stack.controller, stack.currentP, stack.shellSnap, pf.NeedConfigLLM, stack.rt)
	return loop.run()
}
