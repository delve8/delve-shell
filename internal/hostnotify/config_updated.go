package hostnotify

import "sync"

// Send side for hostloop ConfigUpdatedChan (same instance as hostloop.Deps / cli/run wiring).
var (
	configUpdatedMu sync.RWMutex
	configUpdatedC  chan<- struct{}
)

// SetConfigUpdatedChan wires config save/reload notifications to hostloop (runner invalidate + ConfigReloadedMsg).
func SetConfigUpdatedChan(c chan<- struct{}) {
	configUpdatedMu.Lock()
	defer configUpdatedMu.Unlock()
	configUpdatedC = c
}

// NotifyConfigUpdated signals that config or allowlist changed; non-blocking (drops if channel full).
func NotifyConfigUpdated() {
	configUpdatedMu.RLock()
	ch := configUpdatedC
	configUpdatedMu.RUnlock()
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}
