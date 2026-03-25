// Package approvalexec maps approval-flow decisions to HIL side effects (channels, clipboard flags).
// Rendering of decision lines stays in ui (lipgloss styles live there).
package approvalexec

import (
	"delve-shell/internal/agent"
	"delve-shell/internal/approvalflow"
	"delve-shell/internal/approvalview"
)

// Outcome describes what ui should do after a pending approval/sensitive decision.
type Outcome struct {
	LinesKind approvalview.DecisionKind

	HasSensitiveSend bool
	SensitiveChoice  agent.SensitiveChoice

	HasApprovalSend  bool
	ApprovalResponse agent.ApprovalResponse

	ClearSensitive bool
	ClearApproval  bool

	WaitingForAIClear bool

	// Copy workflow: clipboard + suggested line + hint (ui renders hint text).
	DoCopyWorkflow bool
	CopyCommand    string
}

// OutcomeForDecision maps a keyboard decision to channel writes and UI follow-up flags.
// Returns ok=false when the decision should be treated as a no-op (unknown).
func OutcomeForDecision(d approvalflow.Decision, pending *agent.ApprovalRequest, pendingSens *agent.SensitiveConfirmationRequest) (Outcome, bool) {
	switch d {
	case approvalflow.DecisionSensitiveRefuse:
		return Outcome{
			LinesKind:        approvalview.DecisionSensitiveRefuse,
			HasSensitiveSend: true,
			SensitiveChoice:  agent.SensitiveRefuse,
			ClearSensitive:   true,
		}, true
	case approvalflow.DecisionSensitiveRunStore:
		return Outcome{
			LinesKind:        approvalview.DecisionSensitiveRunStore,
			HasSensitiveSend: true,
			SensitiveChoice:  agent.SensitiveRunAndStore,
			ClearSensitive:   true,
		}, true
	case approvalflow.DecisionSensitiveRunNoStore:
		return Outcome{
			LinesKind:        approvalview.DecisionSensitiveRunNoStore,
			HasSensitiveSend: true,
			SensitiveChoice:  agent.SensitiveRunNoStore,
			ClearSensitive:   true,
		}, true
	case approvalflow.DecisionApprove:
		return Outcome{
			LinesKind:        approvalview.DecisionApprove,
			HasApprovalSend:  true,
			ApprovalResponse: agent.ApprovalResponse{Approved: true, CopyRequested: false},
			ClearApproval:    true,
		}, true
	case approvalflow.DecisionReject:
		return Outcome{
			LinesKind:         approvalview.DecisionReject,
			HasApprovalSend:   true,
			ApprovalResponse:  agent.ApprovalResponse{Approved: false, CopyRequested: false},
			ClearApproval:     true,
			WaitingForAIClear: true,
		}, true
	case approvalflow.DecisionCopy:
		if pending == nil {
			return Outcome{}, false
		}
		return Outcome{
			LinesKind:        approvalview.DecisionReject,
			HasApprovalSend:  true,
			ApprovalResponse: agent.ApprovalResponse{Approved: false, CopyRequested: true},
			ClearApproval:    true,
			DoCopyWorkflow:   true,
			CopyCommand:      pending.Command,
		}, true
	case approvalflow.DecisionDismiss:
		return Outcome{
			LinesKind:         approvalview.DecisionDismiss,
			HasApprovalSend:   true,
			ApprovalResponse:  agent.ApprovalResponse{Approved: false, CopyRequested: false},
			ClearApproval:     true,
			WaitingForAIClear: true,
		}, true
	default:
		return Outcome{}, false
	}
}
