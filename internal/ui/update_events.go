package ui

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

func (m Model) handleConfigReloadedMsg() (Model, tea.Cmd) {
	lang := m.getLang()
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigReloaded))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleAgentReplyMsg(msg AgentReplyMsg) (Model, tea.Cmd) {
	m.Interaction.WaitingForAI = false
	lang := m.getLang()
	if msg.Err != nil {
		if errors.Is(msg.Err, context.Canceled) {
			m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyCancelled))))
		} else if errors.Is(msg.Err, agent.ErrLLMNotConfigured) {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyErrLLMNotConfigured, config.ConfigPath()))))
		} else {
			m.Messages = append(m.Messages, errStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyErrorPrefix)+msg.Err.Error())))
		}
		m.Messages = append(m.Messages, "")
	} else if msg.Reply != "" {
		aiLine := i18n.T(lang, i18n.KeyAILabel) + msg.Reply
		w := m.contentWidth()
		m.Messages = append(m.Messages, wrapString(aiLine, w))
		sepW := m.contentWidth()
		m.Messages = append(m.Messages, separatorStyle.Render(strings.Repeat("─", sepW)))
	}
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSystemNotifyMsg(msg SystemNotifyMsg) (Model, tea.Cmd) {
	if msg.Text != "" {
		w := m.contentWidth()
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(wrapString(msg.Text, w))))
		m.Messages = append(m.Messages, "")
		m = m.RefreshViewport()
	}
	return m, nil
}

func (m Model) handleCommandExecutedMsg(msg CommandExecutedMsg) (Model, tea.Cmd) {
	lang := m.getLang()
	var tag string
	if msg.Direct {
		tag = i18n.T(lang, i18n.KeyRunTagDirect)
	} else if msg.Allowed {
		tag = i18n.T(lang, i18n.KeyRunTagAllowlist)
	} else {
		tag = i18n.T(lang, i18n.KeyRunTagApproved)
	}
	runLine := i18n.T(lang, i18n.KeyRunLabel) + msg.Command + " (" + tag + ")"
	w := m.contentWidth()
	m.Messages = append(m.Messages, execStyle.Render(wrapString(runLine, w)))
	if msg.Sensitive {
		m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyResultSensitive)))
	}
	if msg.Result != "" {
		m.Messages = append(m.Messages, resultStyle.Render(wrapString(msg.Result, w)))
	}
	m.Messages = append(m.Messages, "") // blank line after command output
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleConfigLLMCheckDoneMsg(msg ConfigLLMCheckDoneMsg) (Model, tea.Cmd) {
	m.ConfigLLM.Checking = false
	lang := m.getLang()
	if msg.Err != nil {
		m.ConfigLLM.Error = i18n.Tf(lang, i18n.KeyConfigLLMCheckFailed, msg.Err)
		m.Viewport.SetContent(m.buildContent())
		return m, nil
	}
	m.ConfigLLM.Error = ""
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigSavedLLM))))
	if msg.CorrectedBaseURL != "" {
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigLLMBaseURLAutoCorrected, msg.CorrectedBaseURL))))
	}
	m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigLLMCheckOK))))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	m = m.CloseOverlayVisual()
	m.ConfigLLM.Active = false
	if m.Ports.ConfigUpdatedChan != nil {
		select {
		case m.Ports.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m, nil
}

func (m Model) handleApprovalRequestMsg(msg ApprovalRequestMsg) (Model, tea.Cmd) {
	// When an approval is requested, immediately refresh the viewport so the
	// approval card becomes visible, and scroll to bottom.
	m.Approval.Pending = msg
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSensitiveConfirmationRequestMsg(msg SensitiveConfirmationRequestMsg) (Model, tea.Cmd) {
	// Same as approval: ensure the sensitive confirmation card is visible.
	m.Approval.PendingSensitive = msg
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}
