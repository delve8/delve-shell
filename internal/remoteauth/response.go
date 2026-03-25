// Package remoteauth holds SSH credential payloads exchanged between TUI overlays and host/executor layers.
// It stays free of Bubble Tea and UI styling so host bus and runtime packages do not depend on internal/ui.
package remoteauth

// Response carries user-provided credentials from the remote auth overlay back to the host.
//
// Kind is "password" or "identity" (key file path).
// Username is optional; when set, the executor combines it with the host from Target (e.g. overlay default "root").
type Response struct {
	Target   string
	Username string
	Kind     string
	Password string // password when Kind == "password", or key file path when Kind == "identity"
}
