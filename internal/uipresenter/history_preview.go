package uipresenter

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

const historyPreviewWrapWidth = 72

// plainTranscriptVM renders semantic transcript lines as plain text (no ANSI) for overlay bodies.
func plainTranscriptVM(lines []uivm.Line, lang string, wrapWidth int) string {
	if wrapWidth < 20 {
		wrapWidth = historyPreviewWrapWidth
	}
	var b strings.Builder
	delve := i18n.T(lang, i18n.KeyDelveLabel)
	for _, l := range lines {
		switch l.Kind {
		case uivm.LineBlank:
			b.WriteByte('\n')
		case uivm.LineSeparator:
			sepW := wrapWidth
			if sepW > 2 {
				sepW--
			}
			b.WriteString(strings.Repeat("─", sepW))
			b.WriteByte('\n')
		case uivm.LineUser:
			b.WriteString(textwrap.WrapString(i18n.T(lang, i18n.KeyUserLabel)+l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineAI:
			b.WriteString(textwrap.WrapString(i18n.T(lang, i18n.KeyAILabel)+l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineSystemSuggest:
			b.WriteString(textwrap.WrapString(delve+l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineSystemError:
			b.WriteString(textwrap.WrapString(delve+i18n.T(lang, i18n.KeyErrorPrefix)+l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineExec:
			b.WriteString(textwrap.WrapString(l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineResult:
			b.WriteString(textwrap.WrapString(l.Text, wrapWidth))
			b.WriteByte('\n')
		case uivm.LineSessionBanner:
			b.WriteString(textwrap.WrapString(l.Text, wrapWidth))
			b.WriteByte('\n')
		default:
			b.WriteString(textwrap.WrapString(l.Text, wrapWidth))
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// ShowHistoryPreviewDialog opens the read-only preview overlay; the user confirms with Enter in the TUI.
func (p *Presenter) ShowHistoryPreviewDialog(vmLines []uivm.Line, sessionID, lang string) {
	if p == nil {
		return
	}
	body := plainTranscriptVM(vmLines, lang, historyPreviewWrapWidth)
	if strings.TrimSpace(body) == "" {
		body = i18n.T(lang, i18n.KeyHistoryPreviewEmpty)
	}
	body += "\n\n" + i18n.T(lang, i18n.KeyHistoryPreviewFooter)
	title := i18n.Tf(lang, i18n.KeyHistoryPreviewTitle, sessionID)
	p.Raw(ui.HistoryPreviewOverlayMsg{SessionID: sessionID, Title: title, Content: body})
}

// ApplyHistorySwitchBanner replaces the transcript with a short line after a confirmed history switch.
func (p *Presenter) ApplyHistorySwitchBanner(sessionID, lang string) {
	if p == nil {
		return
	}
	banner := i18n.Tf(lang, i18n.KeyHistorySwitchedTo, sessionID)
	tlines := []uivm.Line{
		{Kind: uivm.LineSessionBanner, Text: banner},
		{Kind: uivm.LineBlank},
	}
	p.TranscriptReplace(tlines)
}
