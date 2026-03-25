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
)

// Line is one semantic transcript line. Rendering and wrapping are owned by internal/ui.
type Line struct {
	Kind LineKind
	Text string
}

