package textwrap

import (
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// WrapString breaks s into lines of at most maxWidth terminal cells.
func WrapString(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	var b strings.Builder
	runes := []rune(s)
	start := 0
	cellWidth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\n' {
			b.WriteString(string(runes[start : i+1]))
			start = i + 1
			cellWidth = 0
			continue
		}
		w := runewidth.RuneWidth(r)
		if cellWidth+w > maxWidth && cellWidth > 0 {
			breakAt := i
			for j := i - 1; j >= start; j-- {
				if unicode.IsSpace(runes[j]) {
					breakAt = j + 1
					break
				}
			}
			b.WriteString(string(runes[start:breakAt]))
			b.WriteByte('\n')
			start = breakAt
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}
			cellWidth = 0
			i = start - 1
			continue
		}
		cellWidth += w
	}
	if start < len(runes) {
		b.WriteString(string(runes[start:]))
	}
	return b.String()
}
