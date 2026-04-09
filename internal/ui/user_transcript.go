package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/textwrap"
)

// formatUserTranscriptLines renders submitted user text with the same prompt as the input
// field (typically "> "). When the whole submitted block is shorter than the short separator,
// pad each row to that separator width so the gray band has a stable shape. When the block is
// already wider than the separator, keep rows content-sized.
func formatUserTranscriptLines(prompt, text string, width int) []string {
	plain := prompt + text
	wrapped := textwrap.WrapString(plain, width)
	lines := strings.Split(wrapped, "\n")
	out := make([]string, 0, len(lines))
	targetWidth := longestUserTranscriptLineWidth(lines)
	sepWidth := shortSeparatorDisplayWidth(width)
	if targetWidth < sepWidth {
		targetWidth = sepWidth
	}
	for _, line := range lines {
		bandWidth := ansi.StringWidth(line)
		if targetWidth <= sepWidth {
			bandWidth = targetWidth
		}
		if width > 0 && bandWidth > width {
			bandWidth = width
		}
		styled := userTranscriptStyle.Width(bandWidth).Render(line)
		if width > 0 && ansi.StringWidth(styled) > width {
			styled = ansi.Truncate(styled, width, "")
		}
		out = append(out, styled)
	}
	return out
}

func longestUserTranscriptLineWidth(lines []string) int {
	longest := 0
	for _, line := range lines {
		if w := ansi.StringWidth(line); w > longest {
			longest = w
		}
	}
	return longest
}

// appendTranscriptUserLines mirrors the previous maininput.AppendUserInputLines behavior:
// optional separator, then wrapped user lines, then a blank row.
func appendTranscriptUserLines(messages []string, prompt, text string, width int) []string {
	messages = appendShortTranscriptSeparator(messages, width)
	messages = append(messages, formatUserTranscriptLines(prompt, text, width)...)
	messages = append(messages, "")
	return messages
}
