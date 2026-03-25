package hostnotify

import "sync"

// Current remote execution mode (same source as hostloop RemoteStatusMsg → UI).
var (
	remoteExecMu     sync.RWMutex
	remoteExecActive bool
	remoteExecLabel  string
)

// SetRemoteExecution updates whether commands run on a remote executor and the header label.
func SetRemoteExecution(active bool, label string) {
	remoteExecMu.Lock()
	defer remoteExecMu.Unlock()
	remoteExecActive = active
	remoteExecLabel = label
}

// RemoteActive reports whether the UI should treat execution as remote.
func RemoteActive() bool {
	remoteExecMu.RLock()
	defer remoteExecMu.RUnlock()
	return remoteExecActive
}

// RemoteLabel returns the remote display label (e.g. "dev (user@host)"); empty when local or unset.
func RemoteLabel() string {
	remoteExecMu.RLock()
	defer remoteExecMu.RUnlock()
	return remoteExecLabel
}
