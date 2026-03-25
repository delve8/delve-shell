package ui

import (
	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/uiflow/choicecard"
	"delve-shell/internal/ui/widget"
	"strings"
)

// View implements tea.Model.
func (m Model) View() string {
	lang := m.getLang()
	sepW := m.layout.Width
	if sepW <= 0 {
		sepW = 40
	}
	sepLine := renderSeparator(sepW)
	header := m.titleLine() + "\n" + sepLine + "\n"

	inChoice := m.hasPendingChoiceCard()
	if m.layout.Height <= minInputLayoutWidth {
		out := header + m.buildContent() + "\n" + m.Input.View()
		out += m.waitingLineBelowInput(lang)
		return out
	}
	// Base viewport height: leave room for header, separator, input line, and slash/choice dropdown (the two lines at bottom are for input + suggestions).
	vh := m.mainViewportHeight()
	m.Viewport.Width = m.layout.Width
	m.Viewport.Height = vh
	out := header
	out += m.Viewport.View()
	out += "\n" + sepLine + "\n"
	out += m.Input.View()
	if inChoice {
		out += m.choiceLinesBelowInput(lang)
	} else {
		out += m.slashDropdownBelowInput(lang)
	}
	out += m.waitingLineBelowInput(lang)

	// Render overlay on top if active.
	if m.Overlay.Active {
		out = m.renderOverlay(out)
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

// titleLine returns the fixed title (Remote + Auto-run + status) for display above the viewport; does not scroll.
func (m Model) titleLine() string {
	lang := m.getLang()
	remotePart := m.titleBarLeadingSegment()
	autoRunStr := i18n.T(lang, i18n.KeyAutoRunListOnly)
	if !m.allowlistAutoRunEnabled() {
		autoRunStr = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	autoRunPart := remotePart + " | " + i18n.T(lang, i18n.KeyAutoRunLabel) + autoRunStr + " | "
	statusStr := i18n.T(lang, m.statusKey())
	return widget.RenderTitleLine(autoRunPart, statusStr, m.titleBarStatus(), widget.TitleLineStyles{
		Base:          titleStyle,
		StatusIdle:    statusIdleStyle,
		StatusRunning: statusRunningStyle,
		StatusPending: pendingActionStyle,
		StatusSuggest: suggestStyle,
	})
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
	for _, p := range overlayContentProviderChain.List() {
		if c, handled := p(m); handled {
			content = c
			break
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

// buildContent returns the scrollable viewport content (messages + pending/suggest cards); title is rendered in View().
func (m Model) buildContent() string {
	var b strings.Builder
	for _, line := range m.messages {
		b.WriteString(line)
		b.WriteString("\n")
	}
	if m.appendApprovalViewportContent(&b) {
		return b.String()
	}
	return b.String()
}

// choiceOption is one line in the choice menu (num 1-based, label for display).
type choiceOption struct {
	Num   int
	Label string
}

// choiceCount returns the number of options when in a choice state (approval 2 or 3, sensitive 3, or session list N).
func choiceCount(m Model) int {
	allowlistAutoRunEnabled := m.allowlistAutoRunEnabled()
	return choicecard.ChoiceCount(m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
}

// getChoiceOptions returns the option list for the current choice state (approval 2 or 3 options / sensitive / session list).
func getChoiceOptions(m Model, lang string) []choiceOption {
	allowlistAutoRunEnabled := m.allowlistAutoRunEnabled()
	opts := choicecard.ChoiceOptions(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
	out := make([]choiceOption, 0, len(opts))
	for _, opt := range opts {
		out = append(out, choiceOption{Num: opt.Num, Label: opt.Label})
	}
	return out
}

// syncInputPlaceholder sets the input placeholder to selection hint (1/2 or 1/2/3) when waiting for choice, else normal placeholder.
func (m *Model) syncInputPlaceholder() {
	lang := m.getLang()
	allowlistAutoRunEnabled := m.allowlistAutoRunEnabled()
	m.Input.Placeholder = choicecard.InputPlaceholder(lang, m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil, allowlistAutoRunEnabled)
}

// appendApprovalViewportContent appends sensitive or standard approval blocks to the viewport.
// Returns true if the viewport body is complete (caller should return b.String()).
func (m Model) appendApprovalViewportContent(b *strings.Builder) bool {
	lines, ok := choicecard.BuildPendingLines(
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
