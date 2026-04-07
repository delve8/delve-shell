package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/hil/approvalview"
	hiltypes "delve-shell/internal/hil/types"
)

// PendingCardStyles maps choice-card line kinds to lipgloss styles for the pending card in the viewport.
type PendingCardStyles struct {
	Header, Exec, Suggest, RiskReadOnly, RiskLow, RiskHigh lipgloss.Style
	ExecAutoSafe, ExecAutoRisk, ExecAutoNeutral            lipgloss.Style
}

// RenderPendingApprovalLines renders approvalview.Build output for the transcript viewport.
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
			if len(line.AutoApproveHL) > 0 && line.Text != "" {
				rendered = renderExecLineAutoApproveHL(line.Text, line.AutoApproveHL, s.ExecAutoSafe, s.ExecAutoRisk, s.ExecAutoNeutral)
			} else {
				rendered = s.Exec.Render(line.Text)
			}
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

func renderExecLineAutoApproveHL(text string, spans []hiltypes.AutoApproveHighlightSpan, safe, risk, neutral lipgloss.Style) string {
	if len(spans) == 0 {
		return text
	}
	var b strings.Builder
	n := len(text)
	for _, sp := range spans {
		if sp.Start < 0 || sp.End > n || sp.Start > sp.End {
			continue
		}
		chunk := text[sp.Start:sp.End]
		switch sp.Kind {
		case hiltypes.AutoApproveHighlightRisk:
			b.WriteString(risk.Render(chunk))
		case hiltypes.AutoApproveHighlightSafe:
			b.WriteString(safe.Render(chunk))
		default:
			b.WriteString(neutral.Render(chunk))
		}
	}
	if b.Len() == 0 {
		return neutral.Render(text)
	}
	return b.String()
}
