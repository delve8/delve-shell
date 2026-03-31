package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uivm"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	activeKey := m.Overlay.Key
	m.Interaction.pendingHistorySwitchID = ""
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
	if m.Overlay.Key == HistoryPreviewOverlayKey && key == "enter" {
		id := m.Interaction.pendingHistorySwitchID
		if id != "" && m.CommandSender != nil && m.CommandSender.Send(hostcmd.SessionSwitch{SessionID: id}) {
			m, cmd := m.closeOverlayCommon(true)
			return m, cmd, true
		}
		return m, nil, true
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
	if m.recenterStartupTitleOnce {
		m.recenterStartupTitleOnce = false
		if len(m.messages) > 0 {
			m.messages[0] = startupTitleLine(m.contentWidth())
		}
	}
	if m.layout.Width > minInputLayoutWidth {
		m.Input.SetWidth(m.layout.Width - minInputLayoutWidth)
		if m.ChoiceCard.offlinePaste != nil {
			m.ChoiceCard.offlinePaste.Paste.SetWidth(m.layout.Width - minInputLayoutWidth)
		}
	}
	m = m.syncInputHeight()
	m = m.syncOfflinePasteHeight()
	if m.hasPendingChoiceCard() && m.layout.Height > minInputLayoutWidth {
		m = m.syncChoiceViewport()
	}
	if m.Overlay.Active {
		m = m.InitOverlayViewport()
	}
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
	if m.ChoiceCard.offlinePaste != nil {
		m.ChoiceCard.offlinePaste.Paste.Blur()
	}
	return m, nil
}

func (m Model) handleFocusMsg() (Model, tea.Cmd) {
	if m.Overlay.Active {
		return m, nil
	}
	if m.ChoiceCard.offlinePaste != nil {
		return m, m.ChoiceCard.offlinePaste.Paste.Focus()
	}
	return m, m.Input.Focus()
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd) {
	if m.Overlay.Active {
		var cmd tea.Cmd
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleTranscriptAppendMsg(msg TranscriptAppendMsg) (Model, tea.Cmd) {
	if msg.ClearWaitingForAI || (m.Interaction.WaitingForAI && transcriptHasSystemError(msg.Lines)) {
		m.Interaction.WaitingForAI = false
	}
	if len(msg.Lines) == 0 {
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	m = m.AppendTranscriptLines(rendered...)
	return m.printTranscriptCmd(false)
}

func (m Model) handleTranscriptReplaceMsg(msg TranscriptReplaceMsg) (Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		m = m.withTranscriptReplaced(nil)
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	m = m.withTranscriptReplaced(rendered)
	return m.printTranscriptCmd(true)
}

func (m Model) handleOverlayShowMsg(msg OverlayShowMsg) (Model, tea.Cmd) {
	if msg.Title == "" && strings.TrimSpace(msg.Content) == "" {
		return m, nil
	}
	m = m.OpenOverlayFeature("", msg.Title, msg.Content)
	m = m.InitOverlayViewport()
	return m, nil
}

func (m Model) handleHistoryPreviewOverlayMsg(msg HistoryPreviewOverlayMsg) (Model, tea.Cmd) {
	fields := strings.Fields(strings.TrimSpace(msg.SessionID))
	if len(fields) == 0 || (msg.Title == "" && strings.TrimSpace(msg.Content) == "") {
		return m, nil
	}
	m.Interaction.pendingHistorySwitchID = fields[0]
	m = m.OpenOverlayFeature(HistoryPreviewOverlayKey, msg.Title, msg.Content)
	m = m.InitOverlayViewport()
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
	m = m.syncChoiceViewport()
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
		case uivm.LineSessionBanner:
			rendered = append(rendered, sessionSwitchedStyle.Render(textwrap.WrapString(l.Text, w)))
		default:
			rendered = append(rendered, textwrap.WrapString(l.Text, w))
		}
	}
	return rendered
}

func transcriptHasSystemError(lines []uivm.Line) bool {
	for _, line := range lines {
		if line.Kind == uivm.LineSystemError {
			return true
		}
	}
	return false
}
