package ui

import (
	"strings"
)

// buildContent returns the scrollable viewport content (messages + pending/suggest cards); title is rendered in View().
func (m Model) buildContent() string {
	var b strings.Builder
	for _, line := range m.Messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.appendApprovalViewportContent(&b) {
		return b.String()
	}
	return b.String()
}
