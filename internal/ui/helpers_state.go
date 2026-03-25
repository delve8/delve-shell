package ui

import "strings"

const (
	minInputLayoutWidth      = 4
	minContentWidthFallback  = 80
	mainViewportPadding      = 10
	minOverlayLayoutWidth    = 4
	minOverlayLayoutHeight   = 6
	maxOverlayViewportHeight = 20
)

// hasPendingApproval reports whether the UI is in approval choice mode.
func (m Model) hasPendingApproval() bool {
	return m.Approval.pending != nil || m.Approval.pendingSensitive != nil
}

// contentWidth returns a safe rendering width with fallback.
func (m Model) contentWidth() int {
	w := m.Layout.Width
	if w <= 0 {
		return minContentWidthFallback
	}
	return w
}

// mainViewportHeight returns the viewport height used by main content.
func (m Model) mainViewportHeight() int {
	vh := m.Layout.Height - mainViewportPadding
	if vh < 1 {
		return 1
	}
	return vh
}

// renderSeparator returns a horizontal separator with provided width.
func renderSeparator(width int) string {
	if width < 1 {
		width = 1
	}
	return separatorStyle.Render(strings.Repeat("─", width))
}
