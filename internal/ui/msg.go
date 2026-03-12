package ui

import "delve-shell/internal/agent"

// ApprovalRequestMsg is a command pending user approval (forwarded from agent to TUI via channel).
type ApprovalRequestMsg = *agent.ApprovalRequest

// SensitiveConfirmationRequestMsg is a command that may access sensitive file(s); user chooses refuse / run+store / run+no store.
type SensitiveConfirmationRequestMsg = *agent.SensitiveConfirmationRequest

// AgentReplyMsg is the agent's reply to the user message.
type AgentReplyMsg struct {
	Reply string
	Err   error
}

// SystemNotifyMsg is a system/tool notification (e.g. connected to remote, switched to local), not from the AI.
type SystemNotifyMsg struct {
	Text string
}

// CommandExecutedMsg carries command execution process and result (allowlist/approved/direct/suggested) for display in the conversation.
type CommandExecutedMsg struct {
	Command   string
	Allowed   bool
	Direct    bool   // true = /run direct execution, no AI
	Result    string
	Sensitive bool   // true = result contains private data, not stored and LLM sees "done"
	Suggested bool   // true = suggest mode, command was not executed (user can copy)
}

// ConfigReloadedMsg notifies the UI that config/allowlist was reloaded; next message will use new config.
type ConfigReloadedMsg struct{}

// SessionSwitchedMsg notifies the UI that the session was switched (/new or /sessions).
// Path is the session file path; UI loads history from it to display (empty file for new session).
type SessionSwitchedMsg struct {
	Path string
}

// RemoteStatusMsg notifies the UI that the executor is local or remote, for header display.
type RemoteStatusMsg struct {
	Active bool   // true = remote, false = local
	Label  string // e.g. "dev (root@1.2.3.4)" or "user@host"
}

// RemoteConnectDoneMsg notifies the UI that a /remote on connection attempt finished (from the add-remote overlay).
// When Success is true, the UI closes the overlay and refocuses; when false, the UI clears the "Connecting..." state (auth overlay may follow).
type RemoteConnectDoneMsg struct {
	Success bool
	Label   string
	Err     string
}

// RemoteAuthPromptMsg asks the user to provide additional credentials (e.g. password) for a remote target,
// or to show a Remote Auth dialog while an automatic connection attempt is in progress (e.g. using a configured key).
type RemoteAuthPromptMsg struct {
	Target              string
	Err                 string
	UseConfiguredIdentity bool // true when connecting immediately with a configured identity file; dialog shows "Connecting..." first
}

// RemoteAuthResponse carries user-provided credentials back to CLI.
// Kind is "password" or "identity" (key file path).
// Username is optional; when set, CLI uses it with host from Target for SSH (e.g. overlay default "root").
type RemoteAuthResponse struct {
	Target   string
	Username string // optional; when set, used with host from Target for user@host
	Kind     string // "password" or "identity"
	Password string // password (when Kind == "password") or key file path (when Kind == "identity")
}

// OverlayCloseMsg closes any active overlay.
type OverlayCloseMsg struct{}

// OverlayShowMsg shows an overlay with the given title and content.
type OverlayShowMsg struct {
	Title   string
	Content string
}

// ConfigLLMCheckDoneMsg is sent when the async LLM check (after save) finishes.
// Err non-nil means check failed; CorrectedBaseURL non-empty means /v1 was added and config was updated.
type ConfigLLMCheckDoneMsg struct {
	Err              error
	CorrectedBaseURL string
}
