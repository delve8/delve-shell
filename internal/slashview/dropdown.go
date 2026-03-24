package slashview

import (
	"fmt"
	"strings"
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
		if o.Path == "" && len(o.Cmd) > cmdWidth {
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
	remainingLines := maxVisible
	rows := make([]Row, 0, maxVisible)
	for i := start; i < end && remainingLines > 0; i++ {
		vi := vis[i]
		opt := opts[vi]
		cmdText := opt.Cmd
		if opt.Path != "" {
			cmdText = "/sessions " + opt.Cmd
		}

		if opt.Path == "" && opt.Desc != "" && maxLineLen > 0 {
			descRunes := []rune(opt.Desc)
			prefixRunes := 3 + cmdWidth + 2
			descFirstW := maxLineLen - prefixRunes
			if descFirstW < 8 {
				rows = append(rows, Row{Text: fmt.Sprintf("%-*s", cmdWidth, cmdText), Highlight: i == hiIdx})
				remainingLines--
				if remainingLines <= 0 {
					break
				}
				indent := strings.Repeat(" ", cmdWidth) + "  "
				descW := maxLineLen - len([]rune("   "+indent))
				if descW < 10 {
					descW = 10
				}
				if descW > len(descRunes) {
					descW = len(descRunes)
				}
				if descW < 1 {
					descW = 1
				}
				for j := 0; j < len(descRunes) && remainingLines > 0; j += descW {
					endJ := j + descW
					if endJ > len(descRunes) {
						endJ = len(descRunes)
					}
					rows = append(rows, Row{Text: indent + string(descRunes[j:endJ])})
					remainingLines--
				}
				continue
			}

			if descFirstW > len(descRunes) {
				descFirstW = len(descRunes)
			}
			firstChunk := string(descRunes[:descFirstW])
			rows = append(rows, Row{
				Text:      fmt.Sprintf("%-*s  %s", cmdWidth, cmdText, firstChunk),
				Highlight: i == hiIdx,
			})
			remainingLines--
			if remainingLines <= 0 {
				break
			}
			rest := descRunes[descFirstW:]
			if len(rest) > 0 {
				indent := strings.Repeat(" ", cmdWidth) + "  "
				descW := maxLineLen - len([]rune("   "+indent))
				if descW < 10 {
					descW = 10
				}
				if descW > len(rest) {
					descW = len(rest)
				}
				if descW < 1 {
					descW = 1
				}
				for j := 0; j < len(rest) && remainingLines > 0; j += descW {
					endJ := j + descW
					if endJ > len(rest) {
						endJ = len(rest)
					}
					rows = append(rows, Row{Text: indent + string(rest[j:endJ])})
					remainingLines--
				}
			}
		} else {
			rows = append(rows, Row{
				Text:      fmt.Sprintf("%-*s", cmdWidth, cmdText),
				Highlight: i == hiIdx,
			})
			remainingLines--
		}
	}
	return rows
}
