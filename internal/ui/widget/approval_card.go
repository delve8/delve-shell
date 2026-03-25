package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/approvalview"
)

// PendingCardStyles maps approvalview line kinds to lipgloss styles for the pending approval / sensitive card in the viewport.
type PendingCardStyles struct {
	Header, Exec, Suggest, RiskReadOnly, RiskLow, RiskHigh lipgloss.Style
}

// RenderPendingApprovalLines renders approvalview.Build lines for the transcript viewport (pending prompt, not post-decision).
func RenderPendingApprovalLines(lines []approvalview.Line, s PendingCardStyles) string {
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	for i, line := range lines {
		rendered := line.Text
		switch line.Kind {
		case approvalview.LineHeader:
			rendered = s.Header.Render(line.Text)
		case approvalview.LineExec:
			rendered = s.Exec.Render(line.Text)
		case approvalview.LineSuggest:
			rendered = s.Suggest.Render(line.Text)
		case approvalview.LineRiskReadOnly:
			rendered = s.RiskReadOnly.Render(line.Text)
		case approvalview.LineRiskLow:
			rendered = s.RiskLow.Render(line.Text)
		case approvalview.LineRiskHigh:
			rendered = s.RiskHigh.Render(line.Text)
		}
		b.WriteString(rendered)
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
