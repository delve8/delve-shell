// Package hiltypes defines messages exchanged between the agent runner and the host UI (approval, sensitive
// confirmation, exec notifications). It is orthogonal to package hil (allowlist and sensitive-path policy).
// Import path: delve-shell/internal/hil/types.
//
// This package is intentionally kept free of Bubble Tea and UI styling so host bus and runtime packages do not
// depend on internal/ui.
package hiltypes

// ApprovalResponse is the user's choice for a pending command: Run, Reject, or Copy (copy to clipboard, do not run).
type ApprovalResponse struct {
	Approved      bool // true = run the command
	CopyRequested bool // true = user chose Copy (do not run; copy to clipboard)
}

// ApprovalRequest is sent to HIL: pending command and response channel.
type ApprovalRequest struct {
	Command    string // command to run
	Summary    string // optional short summary (e.g. from SKILL.md); shown separately from Reason
	Reason     string // AI explanation (why, expected effect); may be empty
	RiskLevel  string // RiskLevel* constants; empty if not provided
	SkillName  string // non-empty when pending command is from run_skill (shown on approval card)
	ResponseCh chan ApprovalResponse
}

// SensitiveChoice is the user's choice when a command may access sensitive path(s).
type SensitiveChoice int

const (
	SensitiveRefuse      SensitiveChoice = iota // 1: refuse, do not run
	SensitiveRunAndStore                        // 2: run, return result to AI, store in history
	SensitiveRunNoStore                         // 3: run, return result to AI, do not store in history
)

// SensitiveConfirmationRequest is sent to HIL when command may access sensitive file(s); user picks Refuse / RunAndStore / RunNoStore.
type SensitiveConfirmationRequest struct {
	Command    string
	ResponseCh chan SensitiveChoice
}

// ExecEvent is emitted after command execution for TUI to show HIL process and result.
type ExecEvent struct {
	Command   string
	Allowed   bool   // matched allowlist, no approval needed
	Result    string // stdout + stderr + exit_code for display
	Sensitive bool   // if true, result contains private data, not stored and LLM sees "done"
	Suggested bool   // if true, command was only suggested (suggest mode), not executed
}

// OfflinePasteResponse is the user's submitted pasted output or cancellation for offline (manual) execution.
type OfflinePasteResponse struct {
	Text      string
	Cancelled bool
}

// OfflinePasteRequest asks the UI to show the command and a paste area; blocks until the user submits or cancels.
type OfflinePasteRequest struct {
	Command    string
	Reason     string
	RiskLevel  string
	ResponseCh chan OfflinePasteResponse
}
