package ui

import (
	"strings"

	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/widget"
	"delve-shell/internal/uiflow/choicecard"
)

// appendApprovalViewportContent appends sensitive or standard approval blocks to the viewport.
// Returns true if the viewport body is complete (caller should return b.String()).
func (m Model) appendApprovalViewportContent(b *strings.Builder) bool {
	lines, ok := choicecard.BuildPendingLines(
		m.getLang(),
		m.contentWidth(),
		m.ChoiceCard.pending,
		m.ChoiceCard.pendingSensitive,
		textwrap.WrapString,
	)
	if !ok {
		return false
	}
	b.WriteString("\n")
	b.WriteString(widget.RenderPendingApprovalLines(lines, widget.PendingCardStyles{
		Header:       approvalHeaderStyle,
		Exec:         execStyle,
		Suggest:      suggestStyle,
		RiskReadOnly: riskReadOnlyStyle,
		RiskLow:      riskLowStyle,
		RiskHigh:     riskHighStyle,
	}))
	return true
}
