package ui

import (
	"strings"

	"github.com/atotto/clipboard"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uiflow/choicecard"
	"delve-shell/internal/uivm"
)

func (m Model) handlePendingChoiceKey(key string) (Model, bool) {
	allowlistAutoRunEnabled := m.Host.AllowlistAutoRunEnabled()
	res := choicecard.EvaluateKey(
		key,
		m.ChoiceCard.pending != nil,
		m.ChoiceCard.pendingSensitive != nil,
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

func (m *Model) appendDecisionLines(decision choicecard.DecisionKind, lang string) {
	lines, ok := choicecard.BuildDecisionLines(lang, m.contentWidth(), m.ChoiceCard.pending, m.ChoiceCard.pendingSensitive, decision, textwrap.WrapString)
	if !ok {
		return
	}
	for _, line := range lines {
		rendered := line.Text
		switch line.Kind {
		case choicecard.LineHeader:
			rendered = approvalHeaderStyle.Render(line.Text)
		case choicecard.LineExec:
			rendered = execStyle.Render(line.Text)
		case choicecard.LineSuggest:
			switch decision {
			case choicecard.DecisionKindApprove:
				if line.Text == i18n.T(lang, i18n.KeyApprovalDecisionApproved) {
					rendered = approvalDecisionApprovedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case choicecard.DecisionKindReject:
				if line.Text == i18n.T(lang, i18n.KeyApprovalDecisionRejected) {
					rendered = approvalDecisionRejectedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case choicecard.DecisionKindSensitiveRefuse, choicecard.DecisionKindSensitiveRunStore, choicecard.DecisionKindSensitiveRunNoStore:
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
		case choicecard.LineRiskReadOnly:
			rendered = riskReadOnlyStyle.Render(line.Text)
		case choicecard.LineRiskLow:
			rendered = riskLowStyle.Render(line.Text)
		case choicecard.LineRiskHigh:
			rendered = riskHighStyle.Render(line.Text)
		}
		m.messages = append(m.messages, rendered)
	}
}

func (m Model) applyApprovalDecision(d choicecard.Decision) (Model, bool) {
	lang := m.getLang()
	switch d {
	case choicecard.DecisionSensitiveRefuse, choicecard.DecisionSensitiveRunStore, choicecard.DecisionSensitiveRunNoStore:
		if m.ChoiceCard.pendingSensitive == nil {
			return m, true
		}
		var kind choicecard.DecisionKind
		var choice uivm.SensitiveChoice
		switch d {
		case choicecard.DecisionSensitiveRunStore:
			kind = choicecard.DecisionKindSensitiveRunStore
			choice = uivm.SensitiveRunAndStore
		case choicecard.DecisionSensitiveRunNoStore:
			kind = choicecard.DecisionKindSensitiveRunNoStore
			choice = uivm.SensitiveRunNoStore
		default:
			kind = choicecard.DecisionKindSensitiveRefuse
			choice = uivm.SensitiveRefuse
		}
		m.appendDecisionLines(kind, lang)
		m = m.RefreshViewport()
		if m.ChoiceCard.pendingSensitive.Respond != nil {
			m.ChoiceCard.pendingSensitive.Respond(choice)
		}
		m.ChoiceCard.pendingSensitive = nil
		return m, true

	case choicecard.DecisionApprove, choicecard.DecisionReject, choicecard.DecisionDismiss, choicecard.DecisionCopy:
		if m.ChoiceCard.pending == nil {
			return m, true
		}
		var kind choicecard.DecisionKind
		resp := uivm.ApprovalResponse{Approved: false, CopyRequested: false}
		waitingClear := false
		doCopy := false
		switch d {
		case choicecard.DecisionApprove:
			kind = choicecard.DecisionKindApprove
			resp.Approved = true
		case choicecard.DecisionReject:
			kind = choicecard.DecisionKindReject
			waitingClear = true
		case choicecard.DecisionDismiss:
			kind = choicecard.DecisionKindDismiss
			waitingClear = true
		case choicecard.DecisionCopy:
			kind = choicecard.DecisionKindReject
			resp.CopyRequested = true
			doCopy = true
		}
		m.appendDecisionLines(kind, lang)
		m = m.RefreshViewport()
		if doCopy {
			_ = clipboard.WriteAll(m.ChoiceCard.pending.Command)
			m.appendSuggestedLine(m.ChoiceCard.pending.Command, lang)
			m.messages = append(m.messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
		}
		if m.ChoiceCard.pending.Respond != nil {
			m.ChoiceCard.pending.Respond(resp)
		}
		m.ChoiceCard.pending = nil
		if waitingClear {
			m.Interaction.WaitingForAI = false
		}
		return m, true
	default:
		return m, true
	}
}
