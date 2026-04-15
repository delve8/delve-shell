// Package hiltypes defines messages exchanged between the agent runner and the host UI (approval, sensitive
// confirmation, exec notifications, optional status lines). It is orthogonal to package hil (allowlist and sensitive-path policy).
// Import path: delve-shell/internal/hil/types.
//
// This package is intentionally kept free of Bubble Tea and UI styling so host bus and runtime packages do not
// depend on internal/ui.
package hiltypes

// ApprovalResponse is the user's choice for a pending command: Run, Dismiss, Copy, or reject with guidance.
type ApprovalResponse struct {
	Approved      bool // true = run the command
	CopyRequested bool // true = user chose Copy (do not run; copy to clipboard)
	Guidance      string
}

// AutoApproveHighlightKind classifies a byte range of the command for approval UI coloring.
type AutoApproveHighlightKind uint8

const (
	// AutoApproveHighlightRisk marks text that does not pass per-segment auto-approve policy (or whole-command failure).
	AutoApproveHighlightRisk AutoApproveHighlightKind = iota
	// AutoApproveHighlightSafe marks a segment that would pass segment-level auto-approve checks in isolation.
	AutoApproveHighlightSafe
	// AutoApproveHighlightNeutral marks separators between segments (e.g. |, &&) or non-segment gaps.
	AutoApproveHighlightNeutral
)

// AutoApproveHighlightSpan is a half-open byte range [Start, End) into the same command string shown on the approval card.
// Reason is set for Risk spans when Kind is AutoApproveHighlightRisk (why auto-approve was not granted); empty otherwise.
type AutoApproveHighlightSpan struct {
	Start, End int
	Kind       AutoApproveHighlightKind
	Reason     string
}

// ApprovalRequest is sent to HIL: pending command and response channel.
type ApprovalRequest struct {
	Command   string // command to run
	Summary   string // optional short summary (e.g. from SKILL.md); shown separately from Reason
	Reason    string // AI explanation (why, expected effect); may be empty
	RiskLevel string // RiskLevel* constants; empty if not provided
	SkillName string // non-empty when pending command is from run_skill (shown on approval card)
	// AutoApproveHighlight optional; when non-empty, UI colors ranges to contrast auto-approve-safe vs risky parts.
	AutoApproveHighlight []AutoApproveHighlightSpan
	ResponseCh           chan ApprovalResponse
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

// CommandExecutionState toggles [EXECUTING] UI and input lock while a shell command runs (agent tools).
type CommandExecutionState struct {
	Active bool
}

// AgentNotify is a short system line for the transcript (e.g. remote skill script sync before run).
type AgentNotify struct {
	Text string
}

// ExecStreamStart is sent before streaming stdout/stderr lines for one command execution.
type ExecStreamStart struct {
	Command   string
	Allowed   bool // matched allowlist, no approval needed
	Suggested bool // run_skill path (tag "suggested"); execute_command uses false
	Direct    bool // /exec direct path; agent tool uses false
}

// ExecStreamLine is one stdout or stderr line while a command runs (newline-split).
type ExecStreamLine struct {
	Line   string
	Stderr bool
}

// ExecEvent is emitted after command execution for TUI to show HIL process and result.
type ExecEvent struct {
	Command       string
	Allowed       bool   // matched allowlist, no approval needed
	Result        string // full stdout+stderr+exit, or exit/footer only when Streamed is true
	Sensitive     bool   // if true, result contains private data, not stored and LLM sees "done"
	Suggested     bool   // if true, command was only suggested (suggest mode), not executed
	OfflineManual bool   // if true, command was manually run by the user and relayed back in offline mode
	// Streamed when true: stdout/stderr were appended incrementally; Result is exit code / error footer only.
	Streamed bool
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
