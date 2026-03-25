package run

import (
	"sync"

	"delve-shell/internal/hostapp"
)

var (
	syncAllowlistMu sync.RWMutex
	syncAllowlistFn func(bool)
)

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

// PublishCancelRequest forwards /cancel to the host controller when wired. Returns false if unwired or buffer full.
func PublishCancelRequest() bool { return hostapp.PublishCancelRequest() }

// PublishShellSnapshot sends transcript lines for /sh return restore. Returns false if unwired or buffer full.
func PublishShellSnapshot(msgs []string) bool { return hostapp.PublishShellSnapshot(msgs) }

// PublishExecDirect sends a direct execution command to the host controller (blocking until the channel accepts).
func PublishExecDirect(cmd string) { hostapp.PublishExecDirect(cmd) }
