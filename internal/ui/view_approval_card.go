package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

// appendApprovalViewportContent appends sensitive or standard approval blocks to the viewport.
// Returns true if the viewport body is complete (caller should return b.String()).
func (m Model) appendApprovalViewportContent(b *strings.Builder) bool {
	lang := m.getLang()
	w := m.contentWidth()

	if m.Approval.PendingSensitive != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)) + "\n")
		b.WriteString(execStyle.Render(wrapString(m.Approval.PendingSensitive.Command, w)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice1)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice2)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice3)))
		return true
	}

	if m.Approval.Pending != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		if sn := strings.TrimSpace(m.Approval.Pending.SkillName); sn != "" {
			line := i18n.Tf(lang, i18n.KeySkillLine, sn)
			b.WriteString(suggestStyle.Render(wrapString(line, w)) + "\n")
		}
		switch m.Approval.Pending.RiskLevel {
		case "read_only":
			line := "[" + i18n.T(lang, i18n.KeyRiskReadOnly) + "] " + m.Approval.Pending.Command
			b.WriteString(riskReadOnlyStyle.Render(wrapString(line, w)) + "\n")
		case "low":
			line := "[" + i18n.T(lang, i18n.KeyRiskLow) + "] " + m.Approval.Pending.Command
			b.WriteString(riskLowStyle.Render(wrapString(line, w)) + "\n")
		case "high":
			line := "[" + i18n.T(lang, i18n.KeyRiskHigh) + "] " + m.Approval.Pending.Command
			b.WriteString(riskHighStyle.Render(wrapString(line, w)) + "\n")
		default:
			b.WriteString(execStyle.Render(wrapString(m.Approval.Pending.Command, w)) + "\n")
		}
		if m.Approval.Pending.Summary != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalSummary)+" "+m.Approval.Pending.Summary) + "\n")
		}
		if m.Approval.Pending.Reason != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Approval.Pending.Reason) + "\n")
		}
		return true
	}

	return false
}
