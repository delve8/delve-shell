package hostnotify

import "sync"

var (
	allowlistMu sync.RWMutex
	allowlistFn func() bool
)

// SetAllowlistAutoRunGetter wires the current allowlist auto-run flag (header + approval choice count).
func SetAllowlistAutoRunGetter(fn func() bool) {
	allowlistMu.Lock()
	defer allowlistMu.Unlock()
	allowlistFn = fn
}

// AllowlistAutoRunEnabled returns true when no getter is set, otherwise the getter result.
func AllowlistAutoRunEnabled() bool {
	allowlistMu.RLock()
	fn := allowlistFn
	allowlistMu.RUnlock()
	if fn == nil {
		return true
	}
	return fn()
}
