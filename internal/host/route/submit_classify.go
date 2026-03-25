package route

import "strings"

// UserSubmitKind classifies text passed to the host via submit channel.
type UserSubmitKind int

const (
	// UserSubmitLLM is normal chat (or any input not handled as a host session command).
	UserSubmitLLM UserSubmitKind = iota
	// UserSubmitNewSession matches `/new` after trim (same as UI submit path).
	UserSubmitNewSession
	// UserSubmitSwitchSession matches `/sessions <id>`; SessionID may be empty (no-op, same as prior host behavior).
	UserSubmitSwitchSession
)

// UserSubmit is the routing decision for one submitted line.
type UserSubmit struct {
	Kind      UserSubmitKind
	SessionID string
}

// ClassifyUserSubmit mirrors hostcontroller submit routing for `/new` and `/sessions`.
// The UI trims input before submit; TrimSpace here keeps classification aligned with that path.
func ClassifyUserSubmit(msg string) UserSubmit {
	msg = strings.TrimSpace(msg)
	if msg == "/new" {
		return UserSubmit{Kind: UserSubmitNewSession}
	}
	if strings.HasPrefix(msg, "/sessions ") {
		id := strings.TrimSpace(strings.TrimPrefix(msg, "/sessions "))
		return UserSubmit{Kind: UserSubmitSwitchSession, SessionID: id}
	}
	return UserSubmit{Kind: UserSubmitLLM}
}
