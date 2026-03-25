package ui

import (
	"strings"

	"github.com/atotto/clipboard"

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
		m.messages = append(m.messages, rendered)
	}
}

func (m Model) applyApprovalDecision(d approvalflow.Decision) (Model, bool) {
	lang := m.getLang()
	switch d {
	case approvalflow.DecisionSensitiveRefuse, approvalflow.DecisionSensitiveRunStore, approvalflow.DecisionSensitiveRunNoStore:
		if m.Approval.pendingSensitive == nil {
			return m, true
		}
		var kind approvalview.DecisionKind
		var choice approvalview.SensitiveChoice
		switch d {
		case approvalflow.DecisionSensitiveRunStore:
			kind = approvalview.DecisionSensitiveRunStore
			choice = approvalview.SensitiveRunAndStore
		case approvalflow.DecisionSensitiveRunNoStore:
			kind = approvalview.DecisionSensitiveRunNoStore
			choice = approvalview.SensitiveRunNoStore
		default:
			kind = approvalview.DecisionSensitiveRefuse
			choice = approvalview.SensitiveRefuse
		}
		m.appendDecisionLines(kind, lang)
		m = m.RefreshViewport()
		if m.Approval.pendingSensitive.Respond != nil {
			m.Approval.pendingSensitive.Respond(choice)
		}
		m.Approval.pendingSensitive = nil
		return m, true

	case approvalflow.DecisionApprove, approvalflow.DecisionReject, approvalflow.DecisionDismiss, approvalflow.DecisionCopy:
		if m.Approval.pending == nil {
			return m, true
		}
		var kind approvalview.DecisionKind
		resp := approvalview.ApprovalResponse{Approved: false, CopyRequested: false}
		waitingClear := false
		doCopy := false
		switch d {
		case approvalflow.DecisionApprove:
			kind = approvalview.DecisionApprove
			resp.Approved = true
		case approvalflow.DecisionReject:
			kind = approvalview.DecisionReject
			waitingClear = true
		case approvalflow.DecisionDismiss:
			kind = approvalview.DecisionDismiss
			waitingClear = true
		case approvalflow.DecisionCopy:
			kind = approvalview.DecisionReject
			resp.CopyRequested = true
			doCopy = true
		}
		m.appendDecisionLines(kind, lang)
		m = m.RefreshViewport()
		if doCopy {
			_ = clipboard.WriteAll(m.Approval.pending.Command)
			m.appendSuggestedLine(m.Approval.pending.Command, lang)
			m.messages = append(m.messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
		}
		if m.Approval.pending.Respond != nil {
			m.Approval.pending.Respond(resp)
		}
		m.Approval.pending = nil
		if waitingClear {
			m.Interaction.WaitingForAI = false
		}
		return m, true
	default:
		return m, true
	}
}
