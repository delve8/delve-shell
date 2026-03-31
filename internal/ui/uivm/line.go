// Package uivm defines UI view-model primitives (lines/blocks) that stay free of Bubble Tea and lipgloss.
package uivm

// LineKind is the semantic kind of a transcript line.
type LineKind int

const (
	LinePlain LineKind = iota
	LineBlank
	LineSeparator

	LineUser
	LineAI
	LineSystemSuggest
	LineSystemError

	LineExec
	LineResult

	// LineSessionBanner is a short UI notice (e.g. session switch), not a normal transcript line:
	// rendered without the "Delve:" prefix and with a distinct style.
	LineSessionBanner
)

// Line is one semantic transcript line. Rendering and wrapping are owned by internal/ui.
type Line struct {
	Kind LineKind
	Text string
}
