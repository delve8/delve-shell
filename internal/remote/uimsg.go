package remote

// ExecutionChangedMsg mirrors host-side remote execution status into the TUI model.
// It is consumed by remoteStateProvider and updates ui.Model remote state.
type ExecutionChangedMsg struct {
	Active  bool   // true = remote SSH executor, false = local executor
	Label   string // e.g. "dev (root@1.2.3.4)" or "user@host"
	Offline bool   // true = /access Offline (manual relay); Active is false when Offline is true
	Issue   string // non-empty when remote is degraded/disconnected (e.g. network lost)
}

// ConnectDoneMsg notifies the TUI that a remote connection attempt finished (from controller), e.g. after /access <target>.
// When Success is true, the UI closes the overlay and refocuses; when false, the UI clears "Connecting..." state.
type ConnectDoneMsg struct {
	Success bool
	Label   string
	Err     string
}

// AuthPromptMsg asks the user to provide additional credentials for a remote target.
type AuthPromptMsg struct {
	Target                string
	Socks5Addr            string
	Err                   string
	UseConfiguredIdentity bool
	HostKeyVerify         bool
	HostKeyFingerprint    string
	HostKeyHost           string
}
