package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/textwrap"
)

// formatUserTranscriptLines renders submitted user text with the same prompt as the input
// field (typically "> ") and a full-width gray background per row. Targets ANSI 256-color terminals.
func formatUserTranscriptLines(prompt, text string, width int) []string {
	plain := prompt + text
	wrapped := textwrap.WrapString(plain, width)
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		styled := userTranscriptStyle.Width(width).Render(line)
		if width > 0 && ansi.StringWidth(styled) > width {
			styled = ansi.Truncate(styled, width, "")
		}
		out = append(out, styled)
	}
	return out
}

// appendTranscriptUserLines mirrors the previous maininput.AppendUserInputLines behavior:
// optional separator, then wrapped user lines, then a blank row.
func appendTranscriptUserLines(messages []string, prompt, text string, width int, sepLine string) []string {
	if len(messages) > 0 && messages[len(messages)-1] != sepLine {
		messages = append(messages, sepLine)
	}
	messages = append(messages, formatUserTranscriptLines(prompt, text, width)...)
	messages = append(messages, "")
	return messages
}
