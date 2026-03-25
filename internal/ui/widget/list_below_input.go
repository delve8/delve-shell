package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListRow is one line in a list under the input (slash suggestions or numeric choice).
type ListRow struct {
	Text      string
	Highlight bool
}

// RenderLinesBelowInput renders a leading newline plus one styled line per row.
// prefix is prepended to each row.Text before applying normal or hi (e.g. "   " for slash, " " for choices).
func RenderLinesBelowInput(prefix string, rows []ListRow, normal, hi lipgloss.Style) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n")
	for _, row := range rows {
		line := prefix + row.Text
		if row.Highlight {
			b.WriteString(hi.Render(line) + "\n")
		} else {
			b.WriteString(normal.Render(line) + "\n")
		}
	}
	return b.String()
}
