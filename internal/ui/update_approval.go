package ui

import (
	"strings"

	"github.com/atotto/clipboard"

	"delve-shell/internal/approvalflow"
	"delve-shell/internal/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uiflow/approvalexec"
)

func (m Model) handlePendingChoiceKey(key string) (Model, bool) {
	allowlistAutoRunEnabled := m.Host.AllowlistAutoRunEnabled()
	res := approvalflow.Evaluate(
		key,
		m.Approval.pending != nil,
		m.Approval.pendingSensitive != nil,
		allowlistAutoRunEnabled,
		m.Interaction.ChoiceIndex,
		choiceCount(m),
	)
	if !res.Handled {
		return m, false
	}
	if res.ChoiceChanged {
		m.Interaction.ChoiceIndex = res.ChoiceIndex
		return m, true
	}
	return m.applyApprovalDecision(res.Decision)
}

func (m *Model) appendDecisionLines(decision approvalview.DecisionKind, lang string) {
	lines, ok := approvalview.BuildDecision(lang, m.contentWidth(), m.Approval.pending, m.Approval.pendingSensitive, decision, textwrap.WrapString)
	if !ok {
		return
	}
	for _, line := range lines {
		rendered := line.Text
		switch line.Kind {
		case approvalview.LineHeader:
			rendered = approvalHeaderStyle.Render(line.Text)
		case approvalview.LineExec:
			rendered = execStyle.Render(line.Text)
		case approvalview.LineSuggest:
			switch decision {
			case approvalview.DecisionApprove:
				if line.Text == i18n.T(lang, i18n.KeyApprovalDecisionApproved) {
					rendered = approvalDecisionApprovedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionReject:
				if line.Text == i18n.T(lang, i18n.KeyApprovalDecisionRejected) {
					rendered = approvalDecisionRejectedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionSensitiveRefuse, approvalview.DecisionSensitiveRunStore, approvalview.DecisionSensitiveRunNoStore:
				if strings.HasPrefix(line.Text, i18n.T(lang, i18n.KeySensitiveChoice1)) ||
					strings.HasPrefix(line.Text, i18n.T(lang, i18n.KeySensitiveChoice2)) ||
					strings.HasPrefix(line.Text, i18n.T(lang, i18n.KeySensitiveChoice3)) {
					rendered = suggestHi.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			default:
				rendered = suggestStyle.Render(line.Text)
			}
		case approvalview.LineRiskReadOnly:
			rendered = riskReadOnlyStyle.Render(line.Text)
		case approvalview.LineRiskLow:
			rendered = riskLowStyle.Render(line.Text)
		case approvalview.LineRiskHigh:
			rendered = riskHighStyle.Render(line.Text)
		}
		m.messages = append(m.messages, rendered)
	}
}

func (m Model) applyApprovalDecision(d approvalflow.Decision) (Model, bool) {
	out, ok := approvalexec.OutcomeForDecision(d, m.Approval.pending, m.Approval.pendingSensitive)
	if !ok {
		return m, true
	}
	lang := m.getLang()
	m.appendDecisionLines(out.LinesKind, lang)
	m = m.RefreshViewport()

	if out.HasSensitiveSend && m.Approval.pendingSensitive != nil {
		m.Approval.pendingSensitive.ResponseCh <- out.SensitiveChoice
		m.Approval.pendingSensitive = nil
	}

	if out.HasApprovalSend && m.Approval.pending != nil {
		if out.DoCopyWorkflow {
			_ = clipboard.WriteAll(out.CopyCommand)
			m.appendSuggestedLine(m.Approval.pending.Command, lang)
			m.messages = append(m.messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
		}
		m.Approval.pending.ResponseCh <- out.ApprovalResponse
		m.Approval.pending = nil
	}

	if out.WaitingForAIClear {
		m.Interaction.WaitingForAI = false
	}
	return m, true
}
