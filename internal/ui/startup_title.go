package ui

import (
	"delve-shell/internal/version"
	"strings"

	"github.com/mattn/go-runewidth"
)

// startupTitleLine returns one transcript line: plain title centered with spaces for termWidth
// (ANSI padding must be spaces only so display width matches layout math).
func startupTitleLine(termWidth int) string {
	if termWidth < 1 {
		termWidth = defaultWidth
	}
	title := "Delve Shell " + uiVersionText()
	if strings.TrimSpace(version.Version) == "" {
		title = "Delve Shell"
	}
	w := runewidth.StringWidth(title)
	if w >= termWidth {
		return startupTitleStyle.Render(title)
	}
	pad := (termWidth - w) / 2
	return strings.Repeat(" ", pad) + startupTitleStyle.Render(title)
}
