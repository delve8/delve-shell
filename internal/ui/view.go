package ui

import (
	"delve-shell/internal/hil/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
	"delve-shell/internal/ui/widget"
	"strings"

	"github.com/mattn/go-runewidth"
)

// View implements tea.Model.
func (m *Model) View() string {
	return m.renderScreenSnapshot()
}

func (m *Model) renderBaseScreen() string {
	bottomBlock := m.renderBottomBlock()
	// Choice cards (approval / sensitive / offline paste) render their body via m.messages + tea.Println,
	// same as chat; View() only draws this bottom chrome. Top padding aligns with printed transcript rows.
	padLines := m.normalModeTopPaddingLines(bottomBlock)
	if padLines <= 0 {
		return bottomBlock
	}
	return strings.Repeat("\n", padLines) + bottomBlock
}

func (m *Model) renderBottomBlock() string {
	sepW := m.layout.Width
	if sepW <= 0 {
		sepW = 40
	}
	sepLine := renderShortSeparator(sepW)
	footer := m.footerLine()

	inChoice := m.hasPendingChoiceCard()
	inputSeparator := ""
	if m.primaryInputLineCount() > 1 {
		inputSeparator = "\n"
	}
	preview := m.renderExecStreamPreviewBlock()
	return preview + sepLine + "\n" + m.primaryInputView() + inputSeparator + m.inputBelowBlock(inChoice) + footer
}

func (m *Model) renderScreenSnapshot() string {
	i18n.SetLang(m.getLang())
	out := m.renderBaseScreen()
	if m.Overlay.Active {
		return m.renderOverlay(out)
	}
	return out
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command string) {
	line := FormatRunTranscriptLine(i18n.T(i18n.KeyRunLineSuggested), command)
	m.appendSemanticTranscriptLines(
		uivm.Line{Kind: uivm.LineExec, Text: line},
		uivm.Line{Kind: uivm.LineHint, Text: i18n.T(i18n.KeySuggestedCopyHint)},
	)
}

func (m *Model) titleBarStatus() widget.TitleBarStatus {
	switch m.statusKey() {
	case i18n.KeyStatusIdle:
		return widget.TitleBarStatusIdle
	case i18n.KeyStatusExecuting:
		return widget.TitleBarStatusExecuting
	case i18n.KeyStatusRunning:
		return widget.TitleBarStatusRunning
	case i18n.KeyStatusWaitingUserInput:
		return widget.TitleBarStatusWaitingUserInput
	case i18n.KeyStatusPendingApproval:
		return widget.TitleBarStatusPendingApproval
	case i18n.KeyStatusSuggest:
		return widget.TitleBarStatusSuggest
	default:
		return widget.TitleBarStatusOther
	}
}

// statusKey returns the i18n key for current state: idle, running, or pending approval.
func (m *Model) statusKey() string {
	if m.ChoiceCard.offlinePaste != nil {
		return i18n.KeyStatusWaitingUserInput
	}
	if m.ChoiceCard.pending != nil || m.ChoiceCard.pendingSensitive != nil {
		return i18n.KeyStatusPendingApproval
	}
	if m.Interaction.CommandExecuting {
		return i18n.KeyStatusExecuting
	}
	if m.Interaction.WaitingForAI {
		return i18n.KeyStatusRunning
	}
	return i18n.KeyStatusIdle
}

func (m *Model) titleBarLeadingSegment() string {
	for _, p := range titleBarFragmentProviderChain.List() {
		if seg, ok := p(m); ok {
			return seg
		}
	}
	return i18n.T(i18n.KeyTitleBarLocal)
}

// footerLine returns the fixed status line (status + remote) for display below the input; does not scroll.
func (m *Model) footerLine() string {
	remotePart, remoteIssue := m.footerRemoteParts()
	statusStr := i18n.T(m.statusKey())
	return widget.RenderFooterBar(m.layout.Width, widget.FooterBarParts{
		Remote:              remotePart,
		RemoteIssue:         remoteIssue,
		AutoRunReserveWidth: 0,
		Status:              statusStr,
		StatusReserveWidth:  footerStatusReserveWidth(),
	}, m.titleBarStatus(), widget.TitleLineStyles{
		Base:          titleStyle,
		RemoteIssue:   remoteIssueStyle,
		StatusIdle:    statusIdleStyle,
		StatusRunning: statusRunningStyle,
		StatusPending: pendingActionStyle,
		StatusSuggest: suggestStyle,
	})
}

func (m *Model) footerRemoteParts() (base string, issue string) {
	if m.Remote.Offline {
		return i18n.T(i18n.KeyRemoteTitleBarOffline), ""
	}
	if m.Remote.Active {
		base = i18n.T(i18n.KeyRemoteTitleBarRemote)
		if lbl := strings.TrimSpace(m.Remote.Label); lbl != "" {
			base += " " + lbl
		}
		return base, strings.TrimSpace(m.Remote.Issue)
	}
	return m.titleBarLeadingSegment(), ""
}

func footerStatusReserveWidth() int {
	statuses := []string{
		i18n.T(i18n.KeyStatusIdle),
		i18n.T(i18n.KeyStatusExecuting),
		i18n.T(i18n.KeyStatusRunning),
		i18n.T(i18n.KeyStatusWaitingUserInput),
		i18n.T(i18n.KeyStatusPendingApproval),
		i18n.T(i18n.KeyStatusSuggest),
	}
	maxW := 0
	for _, s := range statuses {
		if w := runewidth.StringWidth(s); w > maxW {
			maxW = w
		}
	}
	return maxW
}

// overlayBoxMaxWidth is the max width of the overlay box so hint lines (e.g. "Up/Down to move... Esc to cancel.") do not wrap.
const overlayBoxMaxWidth = widget.DefaultOverlayBoxMaxWidth

// renderOverlay draws a centered modal box over the base content.
func (m *Model) renderOverlay(base string) string {
	w := m.layout.Width
	h := m.layout.Height
	if w < 20 || h < 6 {
		return base
	}

	var content string
	if feature, ok := overlayFeatureByKey(m.Overlay.Key); ok && feature.Content != nil {
		if c, handled := feature.Content(m); handled {
			content = c
		}
	}
	if content == "" {
		content = m.Overlay.Viewport.View()
		if strings.TrimSpace(m.Overlay.Footer) != "" {
			sepW := overlayInnerWidth(m.layout.Width)
			content = content + "\n" + renderSeparator(sepW) + "\n" + hintStyle.Render(m.Overlay.Footer)
		}
	}

	out := widget.RenderCenteredModal(w, h, overlayBoxMaxWidth, m.Overlay.Title, content)
	if out == "" {
		return base
	}
	return out
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	if m.ChoiceCard.offlinePaste != nil {
		return
	}
	m.Input.Placeholder = approvalview.InputPlaceholder(m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil)
}
