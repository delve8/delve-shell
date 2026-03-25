package run

import "sync"

// Send sides for the host controller (same channel instances as cli.Run / hostbus input ports).
// Set once from cli/run.go before the UI runs.
var (
	shellSnapMu sync.RWMutex
	shellSnapC  chan<- []string

	cancelReqMu sync.RWMutex
	cancelReqC  chan<- struct{}

	syncAllowlistMu sync.RWMutex
	syncAllowlistFn func(bool)

	execDirectMu sync.RWMutex
	execDirectC  chan<- string
)

// SetShellRequestedChan wires the channel for /sh message snapshot (restore after subshell).
func SetShellRequestedChan(c chan<- []string) {
	shellSnapMu.Lock()
	defer shellSnapMu.Unlock()
	shellSnapC = c
}

// trySendShellSnapshot sends a copy of transcript lines to the CLI. Returns false if unwired or full.
func trySendShellSnapshot(msgs []string) bool {
	shellSnapMu.RLock()
	ch := shellSnapC
	shellSnapMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- msgs:
		return true
	default:
		return false
	}
}

// SetCancelRequestChan wires the channel for /cancel while waiting for AI.
func SetCancelRequestChan(c chan<- struct{}) {
	cancelReqMu.Lock()
	defer cancelReqMu.Unlock()
	cancelReqC = c
}

// trySendCancelRequest notifies the host controller to cancel the in-flight LLM request.
func trySendCancelRequest() bool {
	cancelReqMu.RLock()
	ch := cancelReqC
	cancelReqMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// SetSyncAllowlistAutoRun wires allowlist_auto_run persistence sync (atomic + runner invalidate).
// Call once from cli/run.go before the UI runs.
func SetSyncAllowlistAutoRun(fn func(bool)) {
	syncAllowlistMu.Lock()
	defer syncAllowlistMu.Unlock()
	syncAllowlistFn = fn
}

func invokeSyncAllowlistAutoRun(v bool) {
	syncAllowlistMu.RLock()
	fn := syncAllowlistFn
	syncAllowlistMu.RUnlock()
	if fn != nil {
		fn(v)
	}
}

// SetExecDirectChan wires /run <cmd> execution requests to the host controller.
func SetExecDirectChan(c chan<- string) {
	execDirectMu.Lock()
	defer execDirectMu.Unlock()
	execDirectC = c
}

// sendExecDirect forwards a trimmed command (blocking send, same as prior Ports.ExecDirectChan).
func sendExecDirect(cmd string) {
	if cmd == "" {
		return
	}
	execDirectMu.RLock()
	ch := execDirectC
	execDirectMu.RUnlock()
	if ch == nil {
		return
	}
	ch <- cmd
}
