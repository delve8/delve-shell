package uipresenter

import (
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

// ShowHistoryPreviewDialog opens the read-only preview overlay; the user confirms with Enter in the TUI.
func (p *Presenter) ShowHistoryPreviewDialog(vmLines []uivm.Line, sessionID string) {
	if p == nil {
		return
	}
	title := i18n.Tf(i18n.KeyHistoryPreviewTitle, sessionID)
	p.Raw(ui.HistoryPreviewOverlayMsg{SessionID: sessionID, Title: title, Lines: vmLines})
}

// ApplyHistorySwitchBanner replaces the transcript with a short line after a confirmed history switch.
func (p *Presenter) ApplyHistorySwitchBanner(sessionID string) {
	if p == nil {
		return
	}
	banner := i18n.Tf(i18n.KeyHistorySwitchedTo, sessionID)
	tlines := []uivm.Line{
		{Kind: uivm.LineSessionBanner, Text: banner},
		{Kind: uivm.LineBlank},
	}
	p.TranscriptReplace(tlines)
}
