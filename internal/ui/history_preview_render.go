package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/uivm"
)

func overlayBoxWidth(layoutW int) int {
	if layoutW < 20 {
		return 40
	}
	boxW := layoutW - 8
	if boxW > overlayBoxMaxWidth {
		boxW = overlayBoxMaxWidth
	}
	if boxW < 20 {
		boxW = 20
	}
	return boxW
}

// overlayInnerWidth matches the inner content width of the centered modal
// (see [widget.RenderCenteredModal] and overlay box max width).
func overlayInnerWidth(layoutW int) int {
	boxW := overlayBoxWidth(layoutW)
	inner := boxW - 4
	if inner < 20 {
		inner = 20
	}
	return inner
}

// RenderHistoryPreviewTranscript renders semantic transcript lines like the main transcript
// (user / AI / exec / result styles) but never truncates Run lines: long commands wrap in full.
func RenderHistoryPreviewTranscript(lines []uivm.Line, width int) string {
	if width < 1 {
		width = 1
	}
	rendered := make([]string, 0, len(lines))
	for _, l := range lines {
		switch l.Kind {
		case uivm.LineBlank:
			rendered = append(rendered, "")
		case uivm.LineSeparator:
			rendered = append(rendered, renderSeparator(width))
		case uivm.LineUser:
			rendered = append(rendered, formatUserTranscriptLines(i18n.T(i18n.KeyTranscriptUserPrompt), l.Text, width)...)
		case uivm.LineAI:
			rendered = append(rendered, renderAILineTranscript(l.Text, width)...)
		case uivm.LineHint:
			rendered = append(rendered, hintStyle.Render(textwrap.WrapString(l.Text, width)))
		case uivm.LineSystemSuggest:
			rendered = append(rendered, infoStyle.Render(i18n.T(i18n.KeyInfoLabel)+textwrap.WrapString(l.Text, width)))
		case uivm.LineSystemError:
			rendered = append(rendered, errStyle.Render(i18n.T(i18n.KeyErrorPrefix)+l.Text))
		case uivm.LineExec:
			rendered = append(rendered, execStyle.Render(textwrap.WrapString(l.Text, width)))
		case uivm.LineResult:
			plain := ansi.Strip(strings.ReplaceAll(l.Text, "\r", ""))
			wrapped := textwrap.WrapString(plain, width)
			for _, part := range strings.Split(wrapped, "\n") {
				line := resultStyle.Render(part)
				if width > 0 && ansi.StringWidth(line) > width {
					line = ansi.Truncate(line, width, "")
				}
				rendered = append(rendered, line)
			}
		case uivm.LineSessionBanner:
			rendered = append(rendered, sessionSwitchedStyle.Render(textwrap.WrapString(l.Text, width)))
		default:
			rendered = append(rendered, textwrap.WrapString(l.Text, width))
		}
	}
	return strings.Join(rendered, "\n")
}
