package route

import "delve-shell/internal/uitypes"

// SlashSubmitPayload is a design contract for structured slash intent (§10.8.1, ADR 0001).
// It is not wired into SubmitChan or BridgeInputs yet; see docs/adr/0001-slash-submit-payload.md.
//
type SlashSubmitPayload = uitypes.SlashSubmitPayload
