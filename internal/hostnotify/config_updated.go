package hostnotify

import "delve-shell/internal/hostapp"

// NotifyConfigUpdated signals that config or allowlist changed; non-blocking (drops if channel full).
func NotifyConfigUpdated() {
	hostapp.NotifyConfigUpdated()
}
