package agent

import "delve-shell/internal/agent/hiltypes"

// Type aliases keep imports stable for UI and CLI (hiltypes is the canonical definition).
type (
	ApprovalResponse             = hiltypes.ApprovalResponse
	ApprovalRequest              = hiltypes.ApprovalRequest
	SensitiveChoice              = hiltypes.SensitiveChoice
	SensitiveConfirmationRequest = hiltypes.SensitiveConfirmationRequest
	ExecEvent                    = hiltypes.ExecEvent
)

const (
	SensitiveRefuse      = hiltypes.SensitiveRefuse
	SensitiveRunAndStore = hiltypes.SensitiveRunAndStore
	SensitiveRunNoStore  = hiltypes.SensitiveRunNoStore
)
