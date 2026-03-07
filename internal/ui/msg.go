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
