package ui

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
)

func (m Model) appendOfflinePasteViewportContent(b *strings.Builder) {
	lang := m.getLang()
	s := m.ChoiceCard.offlinePaste
	if s == nil {
		return
	}
	w := m.contentWidth()
	b.WriteString("\n")
	b.WriteString(approvalHeaderStyle.Render(textwrap.WrapString(i18n.T(lang, i18n.KeyOfflinePasteTitle), w)))
	b.WriteString("\n\n")
	b.WriteString(textwrap.WrapString(i18n.T(lang, i18n.KeyOfflinePasteIntro), w))
	b.WriteString("\n\n")
	b.WriteString(suggestStyle.Render(textwrap.WrapString(i18n.T(lang, i18n.KeyOfflinePasteReview), w)))
	b.WriteString("\n\n")
	if rl := strings.TrimSpace(s.RiskLevel); rl != "" {
		switch rl {
		case "read_only":
			b.WriteString(riskReadOnlyStyle.Render(textwrap.WrapString(i18n.T(lang, i18n.KeyRiskReadOnly), w)))
		case "low":
			b.WriteString(riskLowStyle.Render(textwrap.WrapString(i18n.T(lang, i18n.KeyRiskLow), w)))
		case "high":
			b.WriteString(riskHighStyle.Render(textwrap.WrapString(i18n.T(lang, i18n.KeyRiskHigh), w)))
		default:
			b.WriteString(textwrap.WrapString(rl, w))
		}
		b.WriteString("\n\n")
	}
	if r := strings.TrimSpace(s.Reason); r != "" {
		b.WriteString(textwrap.WrapString(i18n.T(lang, i18n.KeyApprovalWhy)+r, w))
		b.WriteString("\n\n")
	}
	b.WriteString(execStyle.Render(textwrap.WrapString(s.Command, w)))
	b.WriteString("\n")
	if fb := strings.TrimSpace(s.copyFeedback); fb != "" {
		b.WriteString("\n")
		b.WriteString(hintStyle.Render(textwrap.WrapString(fb, w)))
		b.WriteString("\n")
	}
}
