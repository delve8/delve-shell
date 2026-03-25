package ui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
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
		m.Messages = append(m.Messages, textwrap.WrapString(aiLine, w))
		m.Messages = append(m.Messages, renderSeparator(w))
	}
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSystemNotifyMsg(msg SystemNotifyMsg) (Model, tea.Cmd) {
	if msg.Text != "" {
		w := m.contentWidth()
		m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(textwrap.WrapString(msg.Text, w))))
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
	m.Messages = append(m.Messages, execStyle.Render(textwrap.WrapString(runLine, w)))
	if msg.Sensitive {
		m.Messages = append(m.Messages, suggestStyle.Render(i18n.T(lang, i18n.KeyResultSensitive)))
	}
	if msg.Result != "" {
		m.Messages = append(m.Messages, resultStyle.Render(textwrap.WrapString(msg.Result, w)))
	}
	m.Messages = append(m.Messages, "") // blank line after command output
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleApprovalRequestMsg(msg ApprovalRequestMsg) (Model, tea.Cmd) {
	// When an approval is requested, immediately refresh the viewport so the
	// approval card becomes visible, and scroll to bottom.
	m.Approval.pending = msg
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSensitiveConfirmationRequestMsg(msg SensitiveConfirmationRequestMsg) (Model, tea.Cmd) {
	// Same as approval: ensure the sensitive confirmation card is visible.
	m.Approval.pendingSensitive = msg
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}
