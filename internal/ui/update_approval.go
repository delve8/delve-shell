package ui

import (
	"strings"

	"github.com/atotto/clipboard"

	"delve-shell/internal/agent"
	"delve-shell/internal/approvalview"
	"delve-shell/internal/i18n"
)

func (m Model) handlePendingChoiceKey(key string) (Model, bool) {
	// Choice / approval handling should take precedence over any other key paths,
	// so tests and runtime behavior are stable even if other UI layers evolve.
	inChoice := m.hasPendingApproval()
	if inChoice {
		n := choiceCount(m)
		if n > 0 {
			if key == "enter" {
				// Treat Enter as selecting current option (1-based)
				key = string(rune('1' + m.Interaction.ChoiceIndex))
			} else if key == "up" || key == "down" {
				if key == "down" {
					m.Interaction.ChoiceIndex = (m.Interaction.ChoiceIndex + 1) % n
				} else {
					m.Interaction.ChoiceIndex = (m.Interaction.ChoiceIndex - 1 + n) % n
				}
				return m, true
			}
		}
	}

	if m.Approval.PendingSensitive != nil {
		lang := m.getLang()
		switch key {
		case "1":
			m.appendDecisionLines(approvalview.DecisionSensitiveRefuse, lang)
			m = m.RefreshViewport()
			m.Approval.PendingSensitive.ResponseCh <- agent.SensitiveRefuse
			m.Approval.PendingSensitive = nil
			return m, true
		case "2":
			m.appendDecisionLines(approvalview.DecisionSensitiveRunStore, lang)
			m = m.RefreshViewport()
			m.Approval.PendingSensitive.ResponseCh <- agent.SensitiveRunAndStore
			m.Approval.PendingSensitive = nil
			return m, true
		case "3":
			m.appendDecisionLines(approvalview.DecisionSensitiveRunNoStore, lang)
			m = m.RefreshViewport()
			m.Approval.PendingSensitive.ResponseCh <- agent.SensitiveRunNoStore
			m.Approval.PendingSensitive = nil
			return m, true
		}
		return m, true
	}
	if m.Approval.Pending != nil {
		lang := m.getLang()
		switch key {
		case "1":
			m.appendDecisionLines(approvalview.DecisionApprove, lang)
			m = m.RefreshViewport()

			m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: true, CopyRequested: false}
			m.Approval.Pending = nil
			return m, true
		case "2":
			m.appendDecisionLines(approvalview.DecisionReject, lang)
			m = m.RefreshViewport()
			threeOptions := m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun()
			if threeOptions {
				// 2 = Copy
				_ = clipboard.WriteAll(m.Approval.Pending.Command)
				m.appendSuggestedLine(m.Approval.Pending.Command, lang)
				m.Messages = append(m.Messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
				m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: true}
			} else {
				m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
				m.Interaction.WaitingForAI = false
			}
			m.Approval.Pending = nil
			return m, true
		case "3":
			m.appendDecisionLines(approvalview.DecisionDismiss, lang)
			m = m.RefreshViewport()
			m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
			m.Approval.Pending = nil
			m.Interaction.WaitingForAI = false
			return m, true
		}
		return m, true
	}

	return m, false
}

func (m *Model) appendDecisionLines(decision approvalview.DecisionKind, lang string) {
	lines, ok := approvalview.BuildDecision(lang, m.contentWidth(), m.Approval.Pending, m.Approval.PendingSensitive, decision, wrapString)
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
