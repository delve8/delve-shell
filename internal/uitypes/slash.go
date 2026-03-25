// Package uitypes holds small UI-facing value types shared across ui, uiregistry, and feature packages.
package uitypes

// SlashSubmitPayload is the contract for structured slash intent relay (ADR 0001).
//
// Invariants when used:
//   - RawLine should be non-empty and typically starts with '/' after TrimSpace.
//   - SlashSelectedIndex refers to the visible slash suggestion list index at Enter, or -1 when not applicable.
//   - Must not replace ClassifyUserSubmit handling for "/new" or "/sessions …" without an explicit migration.
type SlashSubmitPayload struct {
	RawLine            string
	SlashSelectedIndex int
	// InputLine is the raw input buffer when Enter came from the slash early path (handleSlashEnterKey).
	// Empty means the main Enter path (executeMainEnterCommandNoRelay). Required for slashflow.EvaluateSlashEnter.
	InputLine string
}
