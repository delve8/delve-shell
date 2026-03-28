package ui

import (
	"delve-shell/internal/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/widget"
	"strings"

	"github.com/mattn/go-runewidth"
)

// View implements tea.Model.
func (m Model) View() string {
	return m.renderScreenSnapshot()
}

func (m Model) renderBaseScreen() string {
	lang := m.getLang()
	sepW := m.layout.Width
	if sepW <= 0 {
		sepW = 40
	}
	sepLine := renderSeparator(sepW)
	footer := m.footerLine()

	inChoice := m.hasPendingChoiceCard()
	inputSeparator := ""
	if m.Input.LineCount() > 1 {
		inputSeparator = "\n"
	}
	bottomBlock := sepLine + "\n" + m.Input.View() + inputSeparator + m.inputBelowBlock(lang, inChoice) + footer
	if !inChoice {
		padLines := m.normalModeTopPaddingLines(bottomBlock)
		if padLines <= 0 {
			return bottomBlock
		}
		return strings.Repeat("\n", padLines) + bottomBlock
	}
	mainBody := m.mainBodyView()
	if m.layout.Height <= minInputLayoutWidth {
		out := bottomBlock
		if mainBody != "" {
			out = mainBody + out
		}
		return out
	}
	// Base viewport height: leave room for the separator, input line, slash/choice dropdown, and footer below.
	out := bottomBlock
	if mainBody != "" {
		out = mainBody + out
	}
	return out
}

func (m Model) renderScreenSnapshot() string {
	out := m.renderBaseScreen()
	if m.Overlay.Active {
		return m.renderOverlay(out)
	}
	return out
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command, lang string) {
	tag := i18n.T(lang, i18n.KeyRunTagSuggested)
	line := i18n.T(lang, i18n.KeyRunLabel) + command + " (" + tag + ")"
	w := m.contentWidth()
	m.messages = append(
		m.messages,
		execStyle.Render(textwrap.WrapString(line, w)),
		hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopyHint)),
	)
}

func (m Model) titleBarStatus() widget.TitleBarStatus {
	switch m.statusKey() {
	case i18n.KeyStatusIdle:
		return widget.TitleBarStatusIdle
	case i18n.KeyStatusRunning:
		return widget.TitleBarStatusRunning
	case i18n.KeyStatusPendingApproval:
		return widget.TitleBarStatusPendingApproval
	case i18n.KeyStatusSuggest:
		return widget.TitleBarStatusSuggest
	default:
		return widget.TitleBarStatusOther
	}
}

// statusKey returns the i18n key for current state: idle, running, or pending approval.
func (m Model) statusKey() string {
	if m.hasPendingChoiceCard() {
		return i18n.KeyStatusPendingApproval
	}
	if m.Interaction.WaitingForAI {
		return i18n.KeyStatusRunning
	}
	return i18n.KeyStatusIdle
}

func (m Model) titleBarLeadingSegment() string {
	for _, p := range titleBarFragmentProviderChain.List() {
		if seg, ok := p(m); ok {
			return seg
		}
	}
	return "Local"
}

// footerLine returns the fixed status line (status + remote) for display below the input; does not scroll.
func (m Model) footerLine() string {
	lang := m.getLang()
	remotePart := m.titleBarLeadingSegment()
	statusStr := i18n.T(lang, m.statusKey())
	return widget.RenderFooterBar(m.layout.Width, widget.FooterBarParts{
		Remote:              remotePart,
		AutoRunReserveWidth: 0,
		Status:              statusStr,
		StatusReserveWidth:  footerStatusReserveWidth(lang),
	}, m.titleBarStatus(), widget.TitleLineStyles{
		Base:          titleStyle,
		StatusIdle:    statusIdleStyle,
		StatusRunning: statusRunningStyle,
		StatusPending: pendingActionStyle,
		StatusSuggest: suggestStyle,
	})
}

func footerStatusReserveWidth(lang string) int {
	statuses := []string{
		i18n.T(lang, i18n.KeyStatusIdle),
		i18n.T(lang, i18n.KeyStatusRunning),
		i18n.T(lang, i18n.KeyStatusPendingApproval),
		i18n.T(lang, i18n.KeyStatusSuggest),
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
func (m Model) renderOverlay(base string) string {
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
	}

	out := widget.RenderCenteredModal(w, h, overlayBoxMaxWidth, m.Overlay.Title, content)
	if out == "" {
		return base
	}
	return out
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	lang := m.getLang()
	m.Input.Placeholder = approvalview.InputPlaceholder(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil)
}

// appendApprovalViewportContent appends sensitive or standard approval blocks to the viewport.
// Returns true if the viewport body is complete (caller should return b.String()).
func (m Model) appendApprovalViewportContent(b *strings.Builder) bool {
	lines, ok := approvalview.Build(
		m.getLang(),
		m.contentWidth(),
		m.ChoiceCard.pending,
		m.ChoiceCard.pendingSensitive,
		textwrap.WrapString,
	)
	if !ok {
		return false
	}
	b.WriteString("\n")
	b.WriteString(widget.RenderPendingApprovalLines(lines, widget.PendingCardStyles{
		Header:       approvalHeaderStyle,
		Exec:         execStyle,
		Suggest:      suggestStyle,
		RiskReadOnly: riskReadOnlyStyle,
		RiskLow:      riskLowStyle,
		RiskHigh:     riskHighStyle,
	}))
	return true
}
