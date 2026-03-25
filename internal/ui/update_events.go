package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
)

func (m Model) handleConfigReloadedMsg() (Model, tea.Cmd) {
	lang := m.getLang()
	m = m.AppendTranscriptLines(
		suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyConfigReloaded))),
		"",
	)
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleAgentReplyMsg(msg AgentReplyMsg) (Model, tea.Cmd) {
	m.Interaction.WaitingForAI = false
	lang := m.getLang()
	if msg.Cancelled {
		m = m.AppendTranscriptLines(suggestStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyCancelled))))
		m = m.AppendTranscriptLines("")
	} else if msg.ErrText != "" {
		m = m.AppendTranscriptLines(errStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyErrorPrefix) + msg.ErrText)))
		m = m.AppendTranscriptLines("")
	} else if msg.Reply != "" {
		aiLine := i18n.T(lang, i18n.KeyAILabel) + msg.Reply
		w := m.contentWidth()
		m = m.AppendTranscriptLines(
			textwrap.WrapString(aiLine, w),
			renderSeparator(w),
		)
	}
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSystemNotifyMsg(msg SystemNotifyMsg) (Model, tea.Cmd) {
	if msg.Text != "" {
		w := m.contentWidth()
		m = m.AppendTranscriptLines(
			suggestStyle.Render(m.delveMsg(textwrap.WrapString(msg.Text, w))),
			"",
		)
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
	m = m.AppendTranscriptLines(execStyle.Render(textwrap.WrapString(runLine, w)))
	if msg.Sensitive {
		m = m.AppendTranscriptLines(suggestStyle.Render(i18n.T(lang, i18n.KeyResultSensitive)))
	}
	if msg.Result != "" {
		m = m.AppendTranscriptLines(resultStyle.Render(textwrap.WrapString(msg.Result, w)))
	}
	m = m.AppendTranscriptLines("") // blank line after command output
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleApprovalRequestMsg(msg ApprovalRequestMsg) (Model, tea.Cmd) {
	// When an approval is requested, immediately refresh the viewport so the
	// approval card becomes visible, and scroll to bottom.
	m.Approval.pending = msg.Pending
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleSensitiveConfirmationRequestMsg(msg SensitiveConfirmationRequestMsg) (Model, tea.Cmd) {
	// Same as approval: ensure the sensitive confirmation card is visible.
	m.Approval.pendingSensitive = msg.Pending
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}
