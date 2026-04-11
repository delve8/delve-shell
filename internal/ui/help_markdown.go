package ui

import "strings"

// RenderHelpMarkdown renders GitHub-flavored Markdown for the help overlay using the same
// glamour pipeline as AI replies ([renderAILineTranscript]), with a fixed wrap width.
// Avoid raw "<placeholder>" in source: glamour parses HTML and may drop unknown tags; use "{placeholder}" or HTML entities.
func RenderHelpMarkdown(md string, width int) string {
	md = strings.TrimSpace(md)
	if md == "" {
		return ""
	}
	md = uiVersionText() + "\n\n" + md
	if width <= 0 {
		width = minContentWidthFallback
	}
	lines := renderAILineTranscript(md, width)
	return strings.Join(lines, "\n")
}
