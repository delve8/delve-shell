package ui

import (
	"delve-shell/internal/i18n"
)

// statusKey returns the i18n key for current state: idle, running, or pending approval.
func (m Model) statusKey() string {
	if m.hasPendingApproval() {
		return i18n.KeyStatusPendingApproval
	}
	if m.Interaction.WaitingForAI {
		return i18n.KeyStatusRunning
	}
	return i18n.KeyStatusIdle
}

func (m Model) titleBarLeadingSegment() string {
	for _, p := range titleBarFragmentProviders {
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
	if m.Ports.GetAllowlistAutoRun != nil && !m.Ports.GetAllowlistAutoRun() {
		autoRunStr = i18n.T(lang, i18n.KeyAutoRunNone)
	}
	autoRunPart := remotePart + " | " + i18n.T(lang, i18n.KeyAutoRunLabel) + autoRunStr + " | "
	statusStr := i18n.T(lang, m.statusKey())
	// Render status with different colors for idle, running, pending, suggest.
	switch m.statusKey() {
	case i18n.KeyStatusIdle:
		return titleStyle.Render(autoRunPart) + statusIdleStyle.Render(statusStr)
	case i18n.KeyStatusRunning:
		return titleStyle.Render(autoRunPart) + statusRunningStyle.Render(statusStr)
	case i18n.KeyStatusPendingApproval:
		return titleStyle.Render(autoRunPart) + pendingActionStyle.Render(statusStr)
	case i18n.KeyStatusSuggest:
		return titleStyle.Render(autoRunPart) + suggestStyle.Render(statusStr)
	default:
		return titleStyle.Render(autoRunPart + statusStr)
	}
}
