package ui

import (
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/widget"
)

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
