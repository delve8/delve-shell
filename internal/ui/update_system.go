package ui

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uivm"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	activeKey := m.Overlay.Key
	m = m.CloseOverlayVisual()
	if feature, ok := overlayFeatureByKey(activeKey); ok && feature.Close != nil {
		m = feature.Close(m, activeKey)
	}
	if refocusInput {
		m.Input.Focus()
	}
	return m, nil
}

// handleOverlayKey routes key input when overlay is active.
func (m Model) handleOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}
	if feature, ok := overlayFeatureByKey(m.Overlay.Key); ok && feature.Key != nil {
		if m2, cmd, handled := feature.Key(m, key, msg); handled {
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
		m.Input.SetWidth(m.layout.Width - minInputLayoutWidth)
	}
	m = m.syncInputHeight()
	if m.layout.Height > minInputLayoutWidth {
		vh := m.mainViewportHeight()
		m.Viewport.Width = m.layout.Width
		m.Viewport.Height = vh
	}
	m = m.RefreshViewport()
	if m.takeOpenConfigLLMOnFirstLayout() {
		for _, entry := range overlayFeatures() {
			if entry.feature.Startup == nil {
				continue
			}
			if m2, cmd, handled := entry.feature.Startup(m); handled {
				return m2, cmd
			}
		}
	}
	return m.printTranscriptCmd(false)
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
	if m2, handled := m.handleScreenSelectionMouse(msg); handled {
		return m2, nil
	}
	if m.Overlay.Active {
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd
	}
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m Model) handleScreenSelectionMouse(msg tea.MouseMsg) (Model, bool) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, false
		}
		pt, ok := m.visiblePointForMouse(msg.Y, msg.X)
		if !ok {
			return m, false
		}
		m = m.withTranscriptSelection(pt, pt)
		return m, true

	case tea.MouseActionMotion:
		if !m.TranscriptSelection.Active {
			return m, false
		}
		pt, ok := m.visiblePointForMouse(msg.Y, msg.X)
		if !ok {
			return m, false
		}
		m.TranscriptSelection.Focus = pt
		return m, true

	case tea.MouseActionRelease:
		if !m.TranscriptSelection.Active {
			return m, false
		}
		if pt, ok := m.visiblePointForMouse(msg.Y, msg.X); ok {
			m.TranscriptSelection.Focus = pt
		}
		text, ok := m.transcriptSelectionText()
		if ok && text != "" {
			_ = clipboard.WriteAll(text)
		}
		return m, true

	default:
		return m, false
	}
}

func (m Model) handleTranscriptAppendMsg(msg TranscriptAppendMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	m = m.AppendTranscriptLines(rendered...)
	m = m.RefreshViewport()
	return m.printTranscriptCmd(false)
}

func (m Model) handleTranscriptReplaceMsg(msg TranscriptReplaceMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		m = m.withTranscriptReplaced(nil)
		m = m.RefreshViewport()
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	m = m.withTranscriptReplaced(rendered)
	m = m.RefreshViewport()
	return m.printTranscriptCmd(true)
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
