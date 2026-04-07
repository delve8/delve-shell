package uivm

import hiltypes "delve-shell/internal/hil/types"

// PendingApproval is the UI view-model for a command pending user approval.
// Respond is invoked by the TUI when the user chooses.
type PendingApproval struct {
	Command              string
	Summary              string
	Reason               string
	RiskLevel            string
	SkillName            string
	AutoApproveHighlight []hiltypes.AutoApproveHighlightSpan
	Respond              func(ApprovalResponse)
}

// ApprovalResponse is the UI-level approval choice.
type ApprovalResponse struct {
	Approved      bool
	CopyRequested bool
}

// SensitiveChoice is the UI-level choice for sensitive path confirmation.
type SensitiveChoice int

const (
	SensitiveRefuse SensitiveChoice = iota
	SensitiveRunAndStore
	SensitiveRunNoStore
)

// PendingSensitive is the UI view-model for a sensitive confirmation prompt.
// Respond is invoked by the TUI when the user chooses.
type PendingSensitive struct {
	Command string
	Respond func(SensitiveChoice)
}
