package widget

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ListRow is one line in a list under the input (slash suggestions or numeric choice).
type ListRow struct {
	Text      string
	Highlight bool
	// PreRendered when true: Text is already a full styled line (e.g. lipgloss output); prefix and normal/hi are not applied.
	PreRendered bool
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
		if row.PreRendered {
			b.WriteString(row.Text + "\n")
			continue
		}
		line := prefix + row.Text
		if row.Highlight {
			b.WriteString(hi.Render(line) + "\n")
		} else {
			b.WriteString(normal.Render(line) + "\n")
		}
	}
	return b.String()
}

// RenderFixedLinesBelowInput renders exactly reserveRows lines below the input.
// Missing rows are padded as blanks so the footer position stays stable.
func RenderFixedLinesBelowInput(prefix string, rows []ListRow, reserveRows int, normal, hi lipgloss.Style) string {
	if reserveRows <= 0 {
		return ""
	}
	if len(rows) > reserveRows {
		rows = rows[:reserveRows]
	}
	padded := make([]ListRow, reserveRows)
	copy(padded, rows)
	return RenderLinesBelowInput(prefix, padded, normal, hi)
}
