package history

import (
	"strings"
)

const sessionSnippetMaxRunes = 60

// FormatSessionSnippetForDisplay turns stored first-turn text into a single-line slash description:
// newlines become the two-character sequence \n, outer space is trimmed, then rune-safe truncation with "...".
func FormatSessionSnippetForDisplay(text string, maxRunes int) string {
	if maxRunes < 12 {
		maxRunes = sessionSnippetMaxRunes
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\n", `\n`)
	text = strings.TrimSpace(text)

	runes := []rune(text)
	ellipsis := "..."
	ell := []rune(ellipsis)
	if len(runes) <= maxRunes {
		return text
	}
	cut := maxRunes - len(ell)
	if cut < 1 {
		return ellipsis
	}
	return string(runes[:cut]) + ellipsis
}
