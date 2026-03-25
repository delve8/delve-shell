package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/uiflow/choicecard"
)

// PendingCardStyles maps choice-card line kinds to lipgloss styles for the pending card in the viewport.
type PendingCardStyles struct {
	Header, Exec, Suggest, RiskReadOnly, RiskLow, RiskHigh lipgloss.Style
}

// RenderPendingApprovalLines renders choicecard.BuildPendingLines output for the transcript viewport.
func RenderPendingApprovalLines(lines []choicecard.Line, s PendingCardStyles) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, line := range lines {
		rendered := line.Text
		switch line.Kind {
		case choicecard.LineHeader:
			rendered = s.Header.Render(line.Text)
		case choicecard.LineExec:
			rendered = s.Exec.Render(line.Text)
		case choicecard.LineSuggest:
			rendered = s.Suggest.Render(line.Text)
		case choicecard.LineRiskReadOnly:
			rendered = s.RiskReadOnly.Render(line.Text)
		case choicecard.LineRiskLow:
			rendered = s.RiskLow.Render(line.Text)
		case choicecard.LineRiskHigh:
			rendered = s.RiskHigh.Render(line.Text)
		}
		b.WriteString(rendered)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
