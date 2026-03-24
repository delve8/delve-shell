package ui

import (
	"strings"

	"github.com/atotto/clipboard"

	"delve-shell/internal/agent"
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
			// Persist a static summary of the sensitive confirmation card and user's choice.
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.Approval.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice1)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Approval.PendingSensitive.ResponseCh <- agent.SensitiveRefuse
			m.Approval.PendingSensitive = nil
			return m, true
		case "2":
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.Approval.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice2)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Approval.PendingSensitive.ResponseCh <- agent.SensitiveRunAndStore
			m.Approval.PendingSensitive = nil
			return m, true
		case "3":
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.Approval.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice3)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
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
			// Persist a static summary of the approval card and user's decision.
			riskLabel := ""
			switch m.Approval.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Approval.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.contentWidth()
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Approval.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				approvalDecisionApprovedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionApproved)),
			)
			if m.Approval.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Approval.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Approval.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Approval.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()

			m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: true, CopyRequested: false}
			m.Approval.Pending = nil
			return m, true
		case "2":
			riskLabel := ""
			switch m.Approval.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Approval.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.contentWidth()
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Approval.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				approvalDecisionRejectedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionRejected)),
			)
			if m.Approval.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Approval.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Approval.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Approval.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
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
			// Only when 3 options: Dismiss
			riskLabel := ""
			switch m.Approval.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Approval.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.contentWidth()
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Approval.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				suggestStyle.Render(i18n.T(lang, i18n.KeyChoiceDismiss)),
			)
			if m.Approval.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Approval.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Approval.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Approval.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Approval.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
			m.Approval.Pending = nil
			m.Interaction.WaitingForAI = false
			return m, true
		}
		return m, true
	}

	return m, false
}
