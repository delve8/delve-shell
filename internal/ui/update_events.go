package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uivm"
)

func (m Model) handleTranscriptAppendMsg(msg TranscriptAppendMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		return m, nil
	}
	lang := m.getLang()
	w := m.contentWidth()
	rendered := make([]string, 0, len(msg.Lines))
	for _, l := range msg.Lines {
		switch l.Kind {
		case uivm.LineBlank:
			rendered = append(rendered, "")
		case uivm.LineSeparator:
			rendered = append(rendered, renderSeparator(w))
		case uivm.LineUser:
			rendered = append(rendered, textwrap.WrapString(i18n.T(lang, i18n.KeyUserLabel)+l.Text, w))
		case uivm.LineAI:
			rendered = append(rendered, textwrap.WrapString(i18n.T(lang, i18n.KeyAILabel)+l.Text, w))
		case uivm.LineSystemSuggest:
			rendered = append(rendered, suggestStyle.Render(m.delveMsg(textwrap.WrapString(l.Text, w))))
		case uivm.LineSystemError:
			rendered = append(rendered, errStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyErrorPrefix)+l.Text)))
		case uivm.LineExec:
			rendered = append(rendered, execStyle.Render(textwrap.WrapString(l.Text, w)))
		case uivm.LineResult:
			rendered = append(rendered, resultStyle.Render(textwrap.WrapString(l.Text, w)))
		default:
			rendered = append(rendered, textwrap.WrapString(l.Text, w))
		}
	}
	m = m.AppendTranscriptLines(rendered...)
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleTranscriptReplaceMsg(msg TranscriptReplaceMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		m = m.WithTranscriptLines(nil)
		m = m.RefreshViewport()
		return m, nil
	}
	// Reuse append renderer but without appending.
	lang := m.getLang()
	w := m.contentWidth()
	rendered := make([]string, 0, len(msg.Lines))
	for _, l := range msg.Lines {
		switch l.Kind {
		case uivm.LineBlank:
			rendered = append(rendered, "")
		case uivm.LineSeparator:
			rendered = append(rendered, renderSeparator(w))
		case uivm.LineUser:
			rendered = append(rendered, textwrap.WrapString(i18n.T(lang, i18n.KeyUserLabel)+l.Text, w))
		case uivm.LineAI:
			rendered = append(rendered, textwrap.WrapString(i18n.T(lang, i18n.KeyAILabel)+l.Text, w))
		case uivm.LineSystemSuggest:
			rendered = append(rendered, suggestStyle.Render(m.delveMsg(textwrap.WrapString(l.Text, w))))
		case uivm.LineSystemError:
			rendered = append(rendered, errStyle.Render(m.delveMsg(i18n.T(lang, i18n.KeyErrorPrefix)+l.Text)))
		case uivm.LineExec:
			rendered = append(rendered, execStyle.Render(textwrap.WrapString(l.Text, w)))
		case uivm.LineResult:
			rendered = append(rendered, resultStyle.Render(textwrap.WrapString(l.Text, w)))
		default:
			rendered = append(rendered, textwrap.WrapString(l.Text, w))
		}
	}
	m = m.WithTranscriptLines(rendered)
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleChoiceCardShowMsg(msg ChoiceCardShowMsg) (Model, tea.Cmd) {
	// Immediately refresh viewport so the card becomes visible and scrolls to bottom.
	if msg.PendingSensitive != nil {
		m.ChoiceCard.pendingSensitive = msg.PendingSensitive
		m.ChoiceCard.pending = nil
	} else if msg.PendingApproval != nil {
		m.ChoiceCard.pending = msg.PendingApproval
		m.ChoiceCard.pendingSensitive = nil
	} else {
		return m, nil
	}
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m = m.RefreshViewport()
	return m, nil
}
