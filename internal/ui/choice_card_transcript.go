package ui

import (
	"strings"

	"delve-shell/internal/hil/approvalview"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/widget"
)

// appendPendingChoiceCardToMessages renders the current approval or sensitive card into styled lines
// and appends them to m.messages (same pipeline as chat transcript + tea.Println).
func (m *Model) appendPendingChoiceCardToMessages() {
	lines, ok := approvalview.Build(
		m.contentWidth(),
		m.ChoiceCard.pending,
		m.ChoiceCard.pendingSensitive,
		textwrap.WrapString,
	)
	if !ok {
		return
	}
	rendered := widget.RenderPendingApprovalLines(lines, widget.PendingCardStyles{
		Header:          approvalHeaderStyle,
		Exec:            execStyle,
		Suggest:         suggestStyle,
		RiskReadOnly:    riskReadOnlyStyle,
		RiskLow:         riskLowStyle,
		RiskHigh:        riskHighStyle,
		ExecAutoSafe:    execAutoSafeStyle,
		ExecAutoRisk:    execAutoRiskStyle,
		ExecAutoNeutral: execAutoNeutralStyle,
		MetaLabel:       metaLabelStyle,
		MetaDetail:      metaDetailStyle,
	})
	m.appendRenderedLinesToMessages(rendered)
}

func (m *Model) appendRenderedLinesToMessages(rendered string) {
	if rendered == "" {
		return
	}
	// One tea.Println per physical row. Do not ansi.Hardwrap here: approval exec lines use lipgloss
	// multi-span styling (auto-approve highlight); Hardwrap can break ANSI and interleave with later rows.
	for _, line := range strings.Split(rendered, "\n") {
		m.messages = append(m.messages, line)
	}
}
