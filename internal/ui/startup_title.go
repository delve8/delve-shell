package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

const startupTitlePlain = "Delve Shell"

// startupTitleLine returns one transcript line: plain title centered with spaces for termWidth
// (ANSI padding must be spaces only so display width matches layout math).
func startupTitleLine(termWidth int) string {
	if termWidth < 1 {
		termWidth = defaultWidth
	}
	w := runewidth.StringWidth(startupTitlePlain)
	if w >= termWidth {
		return startupTitleStyle.Render(startupTitlePlain)
	}
	pad := (termWidth - w) / 2
	return strings.Repeat(" ", pad) + startupTitleStyle.Render(startupTitlePlain)
}
