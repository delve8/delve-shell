package ui

import (
	"strings"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
)

// appendOfflinePasteCardToMessages appends the offline-paste prompt block to m.messages so it
// prints with the same transcript pipeline as chat and approval cards.
func (m *Model) appendOfflinePasteCardToMessages() {
	var b strings.Builder
	m.writeOfflinePasteCardBody(&b)
	m.appendRenderedLinesToMessages(b.String())
}

// writeOfflinePasteCardBody writes styled offline-paste card text, including clipboard feedback
// when available.
func (m *Model) writeOfflinePasteCardBody(b *strings.Builder) {
	s := m.ChoiceCard.offlinePaste
	if s == nil {
		return
	}
	w := m.contentWidth()
	b.WriteString("\n")
	b.WriteString(approvalHeaderStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyOfflinePasteTitle), w)))
	b.WriteString("\n\n")
	b.WriteString(textwrap.WrapString(i18n.T(i18n.KeyOfflinePasteIntro), w))
	b.WriteString("\n\n")
	b.WriteString(suggestStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyOfflinePasteReview), w)))
	b.WriteString("\n\n")
	if rl := strings.TrimSpace(s.RiskLevel); rl != "" {
		switch rl {
		case hiltypes.RiskLevelReadOnly:
			b.WriteString(riskReadOnlyStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyRiskReadOnly), w)))
		case hiltypes.RiskLevelLow:
			b.WriteString(riskLowStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyRiskLow), w)))
		case hiltypes.RiskLevelHigh:
			b.WriteString(riskHighStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyRiskHigh), w)))
		default:
			b.WriteString(textwrap.WrapString(rl, w))
		}
		b.WriteString("\n\n")
	}
	if r := strings.TrimSpace(s.Reason); r != "" {
		b.WriteString(textwrap.WrapString(i18n.T(i18n.KeyApprovalWhy)+r, w))
		b.WriteString("\n\n")
	}
	b.WriteString(execStyle.Render(textwrap.WrapString(offlineCommandReviewText(s.Command, w), w)))
	if fb := strings.TrimSpace(s.copyFeedback); fb != "" {
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render(textwrap.WrapString(fb, w)))
	}
	b.WriteString("\n")
}

func offlineCommandReviewText(command string, width int) string {
	if strings.Contains(command, "\n") {
		return command
	}
	if width > 0 && len(command) <= width {
		return command
	}
	formatted, changed := breakShellControlOperatorsForReview(command)
	if !changed {
		return command
	}
	return formatted
}

func breakShellControlOperatorsForReview(command string) (string, bool) {
	var b strings.Builder
	inSingle := false
	inDouble := false
	escapeDouble := false
	changed := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			b.WriteByte(c)
			if escapeDouble {
				escapeDouble = false
				continue
			}
			if c == '\\' {
				escapeDouble = true
				continue
			}
			if c == '"' {
				inDouble = false
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
			b.WriteByte(c)
		case '"':
			inDouble = true
			b.WriteByte(c)
		case '&':
			if i+1 < len(command) && command[i+1] == '&' {
				b.WriteString("&&\n  ")
				i = skipFollowingSpaces(command, i+1)
				changed = true
			} else {
				b.WriteByte(c)
			}
		case '|':
			if i+1 < len(command) && command[i+1] == '|' {
				b.WriteString("||\n  ")
				i = skipFollowingSpaces(command, i+1)
			} else {
				b.WriteString("|\n  ")
				i = skipFollowingSpaces(command, i)
			}
			changed = true
		case ';':
			b.WriteString(";\n  ")
			i = skipFollowingSpaces(command, i)
			changed = true
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), changed
}

func skipFollowingSpaces(s string, operatorEnd int) int {
	i := operatorEnd
	for i+1 < len(s) && (s[i+1] == ' ' || s[i+1] == '\t') {
		i++
	}
	return i
}
