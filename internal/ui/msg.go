package ui

import "delve-shell/internal/agent"

// ApprovalRequestMsg is a command pending user approval (forwarded from agent to TUI via channel).
type ApprovalRequestMsg = *agent.ApprovalRequest

// AgentReplyMsg is the agent's reply to the user message.
type AgentReplyMsg struct {
	Reply string
	Err   error
}

// CommandExecutedMsg carries command execution process and result (allowlist/approved/direct) for display in the conversation.
type CommandExecutedMsg struct {
	Command   string
	Allowed   bool
	Direct    bool   // true = /run direct execution, no AI
	Result    string
	Sensitive bool   // true = result contains private data, not stored and LLM sees "done"
}

// ConfigReloadedMsg notifies the UI that config/allowlist was reloaded; next message will use new config.
type ConfigReloadedMsg struct{}
