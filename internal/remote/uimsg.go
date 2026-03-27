package remote

// ExecutionChangedMsg mirrors host-side remote execution status into the TUI model.
// It is consumed by remoteStateProvider and updates ui.Model remote state.
type ExecutionChangedMsg struct {
	Active bool   // true = remote, false = local
	Label  string // e.g. "dev (root@1.2.3.4)" or "user@host"
}

// ConnectDoneMsg notifies the TUI that a /remote on connection attempt finished (from controller).
// When Success is true, the UI closes the overlay and refocuses; when false, the UI clears "Connecting..." state.
type ConnectDoneMsg struct {
	Success bool
	Label   string
	Err     string
}

// AuthPromptMsg asks the user to provide additional credentials for a remote target.
type AuthPromptMsg struct {
	Target                string
	Err                   string
	UseConfiguredIdentity bool
	HostKeyVerify         bool
	HostKeyFingerprint    string
	HostKeyHost           string
}

// RunCompletionCacheMsg provides a cached list of candidate strings for /run completion.
// RemoteLabel identifies which remote the list belongs to (empty for local).
type RunCompletionCacheMsg struct {
	RemoteLabel string
	Commands    []string
}
