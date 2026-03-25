package ui

import (
	"strings"

	"github.com/atotto/clipboard"

	"delve-shell/internal/agent"
	"delve-shell/internal/approvalflow"
	"delve-shell/internal/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
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
		m.Messages = append(m.Messages, rendered)
	}
}

func (m Model) applyApprovalDecision(d approvalflow.Decision) (Model, bool) {
	lang := m.getLang()
	switch d {
	case approvalflow.DecisionSensitiveRefuse:
		m.appendDecisionLines(approvalview.DecisionSensitiveRefuse, lang)
		m = m.RefreshViewport()
		m.Approval.pendingSensitive.ResponseCh <- agent.SensitiveRefuse
		m.Approval.pendingSensitive = nil
		return m, true
	case approvalflow.DecisionSensitiveRunStore:
		m.appendDecisionLines(approvalview.DecisionSensitiveRunStore, lang)
		m = m.RefreshViewport()
		m.Approval.pendingSensitive.ResponseCh <- agent.SensitiveRunAndStore
		m.Approval.pendingSensitive = nil
		return m, true
	case approvalflow.DecisionSensitiveRunNoStore:
		m.appendDecisionLines(approvalview.DecisionSensitiveRunNoStore, lang)
		m = m.RefreshViewport()
		m.Approval.pendingSensitive.ResponseCh <- agent.SensitiveRunNoStore
		m.Approval.pendingSensitive = nil
		return m, true
	case approvalflow.DecisionApprove:
		m.appendDecisionLines(approvalview.DecisionApprove, lang)
		m = m.RefreshViewport()
		m.Approval.pending.ResponseCh <- agent.ApprovalResponse{Approved: true, CopyRequested: false}
		m.Approval.pending = nil
		return m, true
	case approvalflow.DecisionReject:
		m.appendDecisionLines(approvalview.DecisionReject, lang)
		m = m.RefreshViewport()
		m.Approval.pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
		m.Approval.pending = nil
		m.Interaction.WaitingForAI = false
		return m, true
	case approvalflow.DecisionCopy:
		m.appendDecisionLines(approvalview.DecisionReject, lang)
		m = m.RefreshViewport()
		_ = clipboard.WriteAll(m.Approval.pending.Command)
		m.appendSuggestedLine(m.Approval.pending.Command, lang)
		m.Messages = append(m.Messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
		m.Approval.pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: true}
		m.Approval.pending = nil
		return m, true
	case approvalflow.DecisionDismiss:
		m.appendDecisionLines(approvalview.DecisionDismiss, lang)
		m = m.RefreshViewport()
		m.Approval.pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
		m.Approval.pending = nil
		m.Interaction.WaitingForAI = false
		return m, true
	default:
		return m, true
	}
}
