package slashview

import (
	"fmt"

	"github.com/mattn/go-runewidth"
)

type Row struct {
	Text      string
	Highlight bool
}

// BuildDropdownRows builds slash dropdown rows without UI style dependencies.
func BuildDropdownRows(opts []Option, vis []int, suggestIndex int, layoutWidth int, maxVisible int) []Row {
	if len(vis) == 0 {
		return nil
	}
	hiIdx := suggestIndex
	if hiIdx >= len(vis) || hiIdx < 0 {
		hiIdx = 0
	}
	start := 0
	if len(vis) > maxVisible {
		start = hiIdx - maxVisible/2
		if start < 0 {
			start = 0
		}
		if start+maxVisible > len(vis) {
			start = len(vis) - maxVisible
		}
	}
	end := start + maxVisible
	if end > len(vis) {
		end = len(vis)
	}

	cmdWidth := 0
	for i := start; i < end; i++ {
		o := opts[vis[i]]
		if len(o.Cmd) > cmdWidth {
			cmdWidth = len(o.Cmd)
		}
	}
	const minCmdWidth = 12
	if cmdWidth < minCmdWidth {
		cmdWidth = minCmdWidth
	}

	maxLineLen := 0
	if layoutWidth > 4 {
		maxLineLen = layoutWidth - 4
	}
	rows := make([]Row, 0, maxVisible)
	for i := start; i < end; i++ {
		vi := vis[i]
		opt := opts[vi]
		line := fmt.Sprintf("%-*s", cmdWidth, opt.Cmd)
		if opt.Desc != "" {
			line += "  " + opt.Desc
		}
		if maxLineLen > 0 {
			line = truncateDropdownLine(line, maxLineLen)
		}
		rows = append(rows, Row{Text: line, Highlight: i == hiIdx})
	}
	return rows
}

func truncateDropdownLine(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return runewidth.Truncate(s, maxWidth, "")
	}
	return runewidth.Truncate(s, maxWidth, "...")
}
