// Package remoteauth holds SSH credential payloads exchanged between TUI overlays and host/executor layers.
// It stays free of Bubble Tea and UI styling so host bus and runtime packages do not depend on internal/ui.
package remoteauth

// Prompt asks the user to provide additional credentials (e.g. password) for a remote target,
// or indicates that an automatic connection attempt is in progress (e.g. using a configured key).
type Prompt struct {
	Target                string
	Socks5Addr            string
	Err                   string
	UseConfiguredIdentity bool // true when connecting immediately with a configured identity file; dialog shows "Connecting..." first
	HostKeyVerify         bool
	HostKeyFingerprint    string
	HostKeyHost           string
}

// Response carries user-provided credentials from the remote auth overlay back to the host.
//
// Kind uses [ResponseKindPassword], [ResponseKindIdentity], [ResponseKindHostKeyAccept], or [ResponseKindHostKeyReject].
// Username is optional; when set, the executor combines it with the host from Target.
type Response struct {
	Target     string
	Socks5Addr string
	Username   string
	Kind       string
	Password   string // password when Kind == "password", or key file path when Kind == "identity"
}
