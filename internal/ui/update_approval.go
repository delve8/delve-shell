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
	inChoice := m.Pending != nil || m.PendingSensitive != nil
	if inChoice {
		n := choiceCount(m)
		if n > 0 {
			if key == "enter" {
				// Treat Enter as selecting current option (1-based)
				key = string(rune('1' + m.ChoiceIndex))
			} else if key == "up" || key == "down" {
				if key == "down" {
					m.ChoiceIndex = (m.ChoiceIndex + 1) % n
				} else {
					m.ChoiceIndex = (m.ChoiceIndex - 1 + n) % n
				}
				return m, true
			}
		}
	}

	if m.PendingSensitive != nil {
		lang := m.getLang()
		switch key {
		case "1":
			// Persist a static summary of the sensitive confirmation card and user's choice.
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice1)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.PendingSensitive.ResponseCh <- agent.SensitiveRefuse
			m.PendingSensitive = nil
			return m, true
		case "2":
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice2)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.PendingSensitive.ResponseCh <- agent.SensitiveRunAndStore
			m.PendingSensitive = nil
			return m, true
		case "3":
			m.Messages = append(m.Messages,
				approvalHeaderStyle.Render(i18n.T(lang, i18n.KeySensitivePrompt)),
				execStyle.Render(m.PendingSensitive.Command),
				suggestHi.Render(i18n.T(lang, i18n.KeySensitiveChoice3)),
			)
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.PendingSensitive.ResponseCh <- agent.SensitiveRunNoStore
			m.PendingSensitive = nil
			return m, true
		}
		return m, true
	}
	if m.Pending != nil {
		lang := m.getLang()
		switch key {
		case "1":
			// Persist a static summary of the approval card and user's decision.
			riskLabel := ""
			switch m.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.Width
			if cmdW <= 0 {
				cmdW = 80
			}
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				approvalDecisionApprovedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionApproved)),
			)
			if m.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()

			m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: true, CopyRequested: false}
			m.Pending = nil
			return m, true
		case "2":
			riskLabel := ""
			switch m.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.Width
			if cmdW <= 0 {
				cmdW = 80
			}
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				approvalDecisionRejectedStyle.Render(i18n.T(lang, i18n.KeyApprovalDecisionRejected)),
			)
			if m.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			threeOptions := m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun()
			if threeOptions {
				// 2 = Copy
				_ = clipboard.WriteAll(m.Pending.Command)
				m.appendSuggestedLine(m.Pending.Command, lang)
				m.Messages = append(m.Messages, hintStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeySuggestedCopied))))
				m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: true}
			} else {
				m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
				m.WaitingForAI = false
			}
			m.Pending = nil
			return m, true
		case "3":
			// Only when 3 options: Dismiss
			riskLabel := ""
			switch m.Pending.RiskLevel {
			case "read_only":
				riskLabel = i18n.T(lang, i18n.KeyRiskReadOnly)
			case "low":
				riskLabel = i18n.T(lang, i18n.KeyRiskLow)
			case "high":
				riskLabel = i18n.T(lang, i18n.KeyRiskHigh)
			}
			commandLine := m.Pending.Command
			if riskLabel != "" {
				commandLine = "[" + riskLabel + "] " + commandLine
			}
			cmdW := m.Width
			if cmdW <= 0 {
				cmdW = 80
			}
			m.Messages = append(m.Messages, approvalHeaderStyle.Render(i18n.T(lang, i18n.KeyApprovalPrompt)))
			if sn := strings.TrimSpace(m.Pending.SkillName); sn != "" {
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(i18n.Tf(lang, i18n.KeySkillLine, sn), cmdW)))
			}
			m.Messages = append(m.Messages,
				execStyle.Render(wrapString(commandLine, cmdW)),
				suggestStyle.Render(i18n.T(lang, i18n.KeyChoiceDismiss)),
			)
			if m.Pending.Summary != "" {
				sumLine := i18n.T(lang, i18n.KeyApprovalSummary) + " " + m.Pending.Summary
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(sumLine, cmdW)))
			}
			if m.Pending.Reason != "" {
				whyLine := i18n.T(lang, i18n.KeyApprovalWhy) + " " + m.Pending.Reason
				m.Messages = append(m.Messages, suggestStyle.Render(wrapString(whyLine, cmdW)))
			}
			m.Viewport.SetContent(m.buildContent())
			m.Viewport.GotoBottom()
			m.Pending.ResponseCh <- agent.ApprovalResponse{Approved: false, CopyRequested: false}
			m.Pending = nil
			m.WaitingForAI = false
			return m, true
		}
		return m, true
	}

	return m, false
}
