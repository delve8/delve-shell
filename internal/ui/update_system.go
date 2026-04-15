package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/teakey"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/uivm"
)

func (m *Model) closeOverlayCommon(refocusInput bool) (*Model, tea.Cmd) {
	activeKey := m.Overlay.Key
	m.Interaction.pendingHistorySwitchID = ""
	m.CloseOverlayVisual()
	if feature, ok := overlayFeatureByKey(activeKey); ok && feature.Close != nil {
		feature.Close(m, activeKey)
	}
	if refocusInput {
		m.Input.Focus()
	}
	return m, nil
}

// handleOverlayKey routes key input when overlay is active.
func (m *Model) handleOverlayKey(key string, msg tea.KeyMsg) (*Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}
	if m.Overlay.Key == HistoryPreviewOverlayKey && key == teakey.Enter {
		id := m.Interaction.pendingHistorySwitchID
		if id != "" && m.CommandSender != nil && m.CommandSender.Send(hostcmd.SessionSwitch{SessionID: id}) {
			m2, cmd := m.closeOverlayCommon(true)
			return m2, cmd, true
		}
		return m, nil, true
	}
	if feature, ok := overlayFeatureByKey(m.Overlay.Key); ok && feature.Key != nil {
		if m2, cmd, handled := feature.Key(m, key, msg); handled {
			return m2, cmd, true
		}
	}

	switch key {
	case teakey.Esc:
		m2, cmd := m.closeOverlayCommon(true)
		return m2, cmd, true
	default:
		var cmd tea.Cmd
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd, true
	}
}

func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
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
		if m.ChoiceCard.approvalGuidance != nil {
			m.ChoiceCard.approvalGuidance.Input.SetWidth(m.layout.Width - minInputLayoutWidth)
		}
	}
	m.syncInputHeight()
	m.syncOfflinePasteHeight()
	m.syncApprovalGuidanceHeight()
	if m.Overlay.Active {
		m.InitOverlayViewport()
	}
	if m.takeOpenConfigModelOnFirstLayout() {
		for _, entry := range overlayFeatures() {
			if entry.feature.Startup == nil {
				continue
			}
			if m2, cmd, handled := entry.feature.Startup(m); handled {
				return m2, cmd
			}
		}
	}
	return m, m.printTranscriptCmd(false)
}

func (m *Model) handleBlurMsg() (*Model, tea.Cmd) {
	m.Input.Blur()
	if m.ChoiceCard.offlinePaste != nil {
		m.ChoiceCard.offlinePaste.Paste.Blur()
	}
	if m.ChoiceCard.approvalGuidance != nil {
		m.ChoiceCard.approvalGuidance.Input.Blur()
	}
	return m, nil
}

func (m *Model) handleFocusMsg() (*Model, tea.Cmd) {
	if m.Overlay.Active {
		return m, nil
	}
	if m.ChoiceCard.offlinePaste != nil {
		return m, m.ChoiceCard.offlinePaste.Paste.Focus()
	}
	if m.ChoiceCard.approvalGuidance != nil {
		return m, m.ChoiceCard.approvalGuidance.Input.Focus()
	}
	return m, m.Input.Focus()
}

func (m *Model) handleMouseMsg(msg tea.MouseMsg) (*Model, tea.Cmd) {
	if m.Overlay.Active {
		var cmd tea.Cmd
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleTranscriptAppendMsg(msg TranscriptAppendMsg) (*Model, tea.Cmd) {
	var focusCmd tea.Cmd
	if msg.ClearWaitingForAI || (m.Interaction.WaitingForAI && transcriptHasSystemError(msg.Lines)) {
		m.Interaction.WaitingForAI = false
		if !m.Overlay.Active && m.ChoiceCard.pending == nil && m.ChoiceCard.pendingSensitive == nil &&
			m.ChoiceCard.offlinePaste == nil && m.ChoiceCard.approvalGuidance == nil && !m.Interaction.CommandExecuting {
			focusCmd = m.Input.Focus()
		}
	}
	if len(msg.Lines) == 0 {
		return m, focusCmd
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	rendered = dropDuplicateRunTranscriptPrefix(m.messages, msg.Lines, rendered)
	if len(rendered) == 0 {
		return m, focusCmd
	}
	m.AppendTranscriptLines(rendered...)
	return m, tea.Batch(focusCmd, m.printTranscriptCmd(false))
}

// transcriptRunLineDedupeLookback is how many prior printed transcript rows to scan when suppressing
// a second identical "Run (...): ..." line (e.g. stream ExecStreamBegin plus a mistaken non-stream ExecEvent).
const transcriptRunLineDedupeLookback = 20

func dropDuplicateRunTranscriptPrefix(messages []string, semantic []uivm.Line, rendered []string) []string {
	if len(rendered) == 0 || len(semantic) == 0 {
		return rendered
	}
	if semantic[0].Kind != uivm.LineExec || !IsRunTranscriptExecLine(semantic[0].Text) {
		return rendered
	}
	if !isRecentDuplicateRunTranscriptLine(messages, rendered[0]) {
		return rendered
	}
	if len(rendered) == 1 {
		return nil
	}
	return rendered[1:]
}

func isRecentDuplicateRunTranscriptLine(messages []string, newRenderedLine string) bool {
	newPlain := ansi.Strip(newRenderedLine)
	start := len(messages) - transcriptRunLineDedupeLookback
	if start < 0 {
		start = 0
	}
	for i := len(messages) - 1; i >= start; i-- {
		prev := messages[i]
		if !IsRunTranscriptExecLine(prev) {
			continue
		}
		if ansi.Strip(prev) == newPlain {
			return true
		}
	}
	return false
}

func (m *Model) handleTranscriptReplaceMsg(msg TranscriptReplaceMsg) (*Model, tea.Cmd) {
	if len(msg.Lines) == 0 {
		m.withTranscriptReplaced(nil)
		return m, nil
	}
	rendered := m.renderTranscriptLines(msg.Lines)
	m.withTranscriptReplaced(rendered)
	return m, m.printTranscriptCmd(true)
}

func (m *Model) handleOverlayShowMsg(msg OverlayShowMsg) (*Model, tea.Cmd) {
	if msg.Title == "" && strings.TrimSpace(msg.Content) == "" {
		return m, nil
	}
	m.OpenOverlayFeature("", msg.Title, msg.Content)
	m.InitOverlayViewport()
	return m, nil
}

func (m *Model) handleHistoryPreviewOverlayMsg(msg HistoryPreviewOverlayMsg) (*Model, tea.Cmd) {
	fields := strings.Fields(strings.TrimSpace(msg.SessionID))
	if len(fields) == 0 || msg.Title == "" {
		return m, nil
	}
	footer := i18n.T(i18n.KeyHistoryPreviewFooter)
	var body string
	if len(msg.Lines) > 0 {
		w := overlayInnerWidth(m.layout.Width)
		body = RenderHistoryPreviewTranscript(msg.Lines, w)
		if strings.TrimSpace(body) == "" {
			body = i18n.T(i18n.KeyHistoryPreviewEmpty)
		}
	} else {
		body = msg.Content
		if strings.TrimSpace(body) == "" {
			body = i18n.T(i18n.KeyHistoryPreviewEmpty)
		}
	}
	m.Interaction.pendingHistorySwitchID = fields[0]
	m.openOverlayFeature(HistoryPreviewOverlayKey, msg.Title, body, footer)
	m.InitOverlayViewport()
	return m, nil
}

func (m *Model) handleChoiceCardShowMsg(msg ChoiceCardShowMsg) (*Model, tea.Cmd) {
	if msg.PendingSensitive != nil {
		m.ChoiceCard.pendingSensitive = msg.PendingSensitive
		m.ChoiceCard.pending = nil
		m.ChoiceCard.approvalGuidance = nil
	} else if msg.PendingApproval != nil {
		m.ChoiceCard.pending = msg.PendingApproval
		m.ChoiceCard.pendingSensitive = nil
		m.ChoiceCard.approvalGuidance = nil
	} else {
		return m, nil
	}
	m.Interaction.ChoiceIndex = 0
	m.syncInputPlaceholder()
	m.appendPendingChoiceCardToMessages()
	return m, m.printTranscriptCmd(false)
}

func (m *Model) renderTranscriptLines(lines []uivm.Line) []string {
	w := m.contentWidth()
	rendered := make([]string, 0, len(lines))
	for _, l := range lines {
		switch l.Kind {
		case uivm.LineBlank:
			rendered = append(rendered, "")
		case uivm.LineSeparator:
			rendered = append(rendered, renderShortSeparator(w))
		case uivm.LineUser:
			rendered = append(rendered, formatUserTranscriptLines(i18n.T(i18n.KeyTranscriptUserPrompt), l.Text, w)...)
		case uivm.LineAI:
			rendered = append(rendered, renderAILineTranscript(l.Text, w)...)
		case uivm.LineHint:
			rendered = append(rendered, hintStyle.Render(textwrap.WrapString(l.Text, w)))
		case uivm.LineSystemSuggest:
			rendered = append(rendered, infoStyle.Render(m.infoMsg(textwrap.WrapString(l.Text, w))))
		case uivm.LineSystemError:
			rendered = append(rendered, errStyle.Render(i18n.T(i18n.KeyErrorPrefix)+l.Text))
		case uivm.LineExec:
			txt := l.Text
			if IsRunTranscriptExecLine(txt) {
				txt = ClampRunTranscriptPlain(txt, RunTranscriptDisplayMaxCells(w))
				rendered = append(rendered, execStyle.Render(txt))
			} else {
				rendered = append(rendered, execStyle.Render(textwrap.WrapString(l.Text, w)))
			}
		case uivm.LineResult:
			// Command/tool stdout may include ANSI (e.g. kubectl color). Bubble Tea queues Println lines
			// without truncating when width >= terminal; the terminal soft-wraps while the renderer still
			// assumes one row, so the next View() redraw can start mid-line and merge with placeholder/footer.
			plain := ansi.Strip(strings.ReplaceAll(l.Text, "\r", ""))
			wrapped := textwrap.WrapString(plain, w)
			for _, part := range strings.Split(wrapped, "\n") {
				line := resultStyle.Render(part)
				if w > 0 && ansi.StringWidth(line) > w {
					line = ansi.Truncate(line, w, "")
				}
				rendered = append(rendered, line)
			}
		case uivm.LineSessionBanner:
			rendered = append(rendered, sessionSwitchedStyle.Render(textwrap.WrapString(l.Text, w)))
		default:
			rendered = append(rendered, textwrap.WrapString(l.Text, w))
		}
	}
	return rendered
}

func (m *Model) appendSemanticTranscriptLines(lines ...uivm.Line) {
	if len(lines) == 0 {
		return
	}
	m.AppendTranscriptLines(m.renderTranscriptLines(lines)...)
}

func transcriptHasSystemError(lines []uivm.Line) bool {
	for _, line := range lines {
		if line.Kind == uivm.LineSystemError {
			return true
		}
	}
	return false
}
