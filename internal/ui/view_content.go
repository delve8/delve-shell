package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

// buildContent returns the scrollable viewport content (messages + pending/suggest cards); title is rendered in View().
func (m Model) buildContent() string {
	lang := m.getLang()
	var b strings.Builder
	for _, line := range m.Messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.PendingSensitive != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)) + "\n")
		w := m.Width
		if w <= 0 {
			w = 80
		}
		b.WriteString(execStyle.Render(wrapString(m.PendingSensitive.Command, w)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice1)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice2)) + "\n")
		b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeySensitiveChoice3)))
		return b.String()
	}
	if m.Pending != nil {
		b.WriteString("\n")
		b.WriteString(approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)) + "\n")
		w := m.Width
		if w <= 0 {
			w = 80
		}
		if sn := strings.TrimSpace(m.Pending.SkillName); sn != "" {
			line := i18n.Tf(lang, i18n.KeySkillLine, sn)
			b.WriteString(suggestStyle.Render(wrapString(line, w)) + "\n")
		}
		switch m.Pending.RiskLevel {
		case "read_only":
			line := "[" + i18n.T(lang, i18n.KeyRiskReadOnly) + "] " + m.Pending.Command
			b.WriteString(riskReadOnlyStyle.Render(wrapString(line, w)) + "\n")
		case "low":
			line := "[" + i18n.T(lang, i18n.KeyRiskLow) + "] " + m.Pending.Command
			b.WriteString(riskLowStyle.Render(wrapString(line, w)) + "\n")
		case "high":
			line := "[" + i18n.T(lang, i18n.KeyRiskHigh) + "] " + m.Pending.Command
			b.WriteString(riskHighStyle.Render(wrapString(line, w)) + "\n")
		default:
			b.WriteString(execStyle.Render(wrapString(m.Pending.Command, w)) + "\n")
		}
		if m.Pending.Summary != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalSummary)+" "+m.Pending.Summary) + "\n")
		}
		if m.Pending.Reason != "" {
			b.WriteString(suggestStyle.Render(i18n.T(lang, i18n.KeyApprovalWhy)+" "+m.Pending.Reason) + "\n")
		}
		return b.String()
	}
	return b.String()
}
