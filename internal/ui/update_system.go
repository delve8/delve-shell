package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uivm"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	m = m.CloseOverlayVisual()
	for _, h := range overlayCloseHookChain.List() {
		m = h(m)
	}
	if refocusInput {
		m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleOverlayShowMsg(msg OverlayShowMsg) (Model, tea.Cmd) {
	m = m.OpenOverlay(msg.Title, msg.Content)
	m = m.InitOverlayViewport()
	return m, nil
}

func (m Model) handleOverlayCloseMsg() (Model, tea.Cmd) {
	return m.closeOverlayCommon(false)
}

// handleOverlayKey routes key input when overlay is active.
func (m Model) handleOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}

	for _, p := range overlayKeyProviderChain.List() {
		if m2, cmd, handled := p(m, key, msg); handled {
			return m2, cmd, true
		}
	}

	switch key {
	case "esc":
		m, cmd := m.closeOverlayCommon(true)
		return m, cmd, true
	default:
		var cmd tea.Cmd
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd, true
	}
}

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.layout.Width = msg.Width
	m.layout.Height = msg.Height
	if m.layout.Width > minInputLayoutWidth {
		m.Input.Width = m.layout.Width - minInputLayoutWidth
	}
	if m.layout.Height > minInputLayoutWidth {
		vh := m.mainViewportHeight()
		m.Viewport.Width = m.layout.Width
		m.Viewport.Height = vh
	}
	m = m.RefreshViewport()
	if m.takeOpenConfigLLMOnFirstLayout() {
		for _, p := range startupOverlayProviderChain.List() {
			if m2, cmd, handled := p(m); handled {
				return m2, cmd
			}
		}
	}
	return m, nil
}

func (m Model) handleBlurMsg() (Model, tea.Cmd) {
	m.Input.Blur()
	return m, nil
}

func (m Model) handleFocusMsg() (Model, tea.Cmd) {
	if !m.Overlay.Active {
		return m, m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m Model) handleTranscriptAppendMsg(msg TranscriptAppendMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
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
	rendered := m.renderTranscriptLines(msg.Lines)
	m = m.WithTranscriptLines(rendered)
	m = m.RefreshViewport()
	return m, nil
}

func (m Model) handleChoiceCardShowMsg(msg ChoiceCardShowMsg) (Model, tea.Cmd) {
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

func (m Model) handleLifecycleSlashExecuteMsg(msg LifecycleSlashExecuteMsg) (Model, tea.Cmd) {
	if msg.InputLine != "" {
		m2, cmd, handled := m.execSlashEnterKeyLocal(msg.InputLine)
		if handled {
			return m2, cmd
		}
		return m.executeMainEnterCommandNoRelay(strings.TrimSpace(msg.InputLine), msg.SelectedIndex)
	}
	return m.executeMainEnterCommandNoRelay(msg.RawText, msg.SelectedIndex)
}

func (m Model) renderTranscriptLines(lines []uivm.Line) []string {
	lang := m.getLang()
	w := m.contentWidth()
	rendered := make([]string, 0, len(lines))
	for _, l := range lines {
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
	return rendered
}
