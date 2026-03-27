package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// ScreenPoint identifies a cell in the rendered screen buffer.
type ScreenPoint struct {
	Row int
	Col int
}

// ScreenBuffer is a plain-text snapshot of the currently rendered screen.
type ScreenBuffer struct {
	Lines []string
}

func newScreenBuffer(rendered string) ScreenBuffer {
	if rendered == "" {
		return ScreenBuffer{}
	}
	rendered = strings.TrimSuffix(rendered, "\n")
	if rendered == "" {
		return ScreenBuffer{}
	}
	lines := strings.Split(rendered, "\n")
	for i, line := range lines {
		lines[i] = transcriptAnsiStrip.ReplaceAllString(line, "")
	}
	return ScreenBuffer{Lines: lines}
}

func (b ScreenBuffer) lineCount() int {
	return len(b.Lines)
}

func (b ScreenBuffer) clampPoint(y, x int) (ScreenPoint, bool) {
	if len(b.Lines) == 0 {
		return ScreenPoint{}, false
	}
	if y < 0 {
		y = 0
	}
	if y >= len(b.Lines) {
		y = len(b.Lines) - 1
	}
	line := b.Lines[y]
	width := runewidth.StringWidth(line)
	if width <= 0 {
		return ScreenPoint{Row: y, Col: 0}, true
	}
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}
	return ScreenPoint{Row: y, Col: x}, true
}

func (b ScreenBuffer) selectionBounds(sel ScreenSelectionState) (ScreenPoint, ScreenPoint, bool) {
	if !sel.Active || len(b.Lines) == 0 {
		return ScreenPoint{}, ScreenPoint{}, false
	}
	start, end := sel.Anchor, sel.Focus
	if start.Row > end.Row || (start.Row == end.Row && start.Col > end.Col) {
		start, end = end, start
	}
	return start, end, true
}

func (b ScreenBuffer) selectionText(sel ScreenSelectionState) (string, bool) {
	start, end, ok := b.selectionBounds(sel)
	if !ok {
		return "", false
	}
	if start.Row < 0 || start.Row >= len(b.Lines) {
		return "", false
	}
	if end.Row >= len(b.Lines) {
		end.Row = len(b.Lines) - 1
	}
	var out []string
	for row := start.Row; row <= end.Row; row++ {
		line := b.Lines[row]
		line = strings.TrimRight(line, " ")
		switch {
		case start.Row == end.Row:
			out = append(out, sliceDisplayColumns(line, start.Col, end.Col))
		case row == start.Row:
			out = append(out, sliceDisplayColumns(line, start.Col, runewidth.StringWidth(line)-1))
		case row == end.Row:
			out = append(out, sliceDisplayColumns(line, 0, end.Col))
		default:
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n"), true
}

func (b ScreenBuffer) renderSelection(sel ScreenSelectionState, highlight lipglossStyleRenderer) string {
	start, end, ok := b.selectionBounds(sel)
	if !ok {
		return strings.Join(b.Lines, "\n")
	}
	var out []string
	for row, line := range b.Lines {
		plain := strings.TrimRight(line, " ")
		if row < start.Row || row > end.Row {
			out = append(out, plain)
			continue
		}
		if start.Row == end.Row {
			out = append(out, renderSelectedSpan(plain, start.Col, end.Col, highlight))
			continue
		}
		switch row {
		case start.Row:
			out = append(out, renderSelectedSpan(plain, start.Col, runewidth.StringWidth(plain)-1, highlight))
		case end.Row:
			out = append(out, renderSelectedSpan(plain, 0, end.Col, highlight))
		default:
			out = append(out, highlight.Render(plain))
		}
	}
	return strings.Join(out, "\n")
}

type lipglossStyleRenderer interface {
	Render(...string) string
}

func renderSelectedSpan(line string, startCol, endCol int, style lipglossStyleRenderer) string {
	if endCol < startCol {
		startCol, endCol = endCol, startCol
	}
	before, selected, after := sliceDisplayColumnsParts(line, startCol, endCol)
	return before + style.Render(selected) + after
}

func sliceDisplayColumns(line string, startCol, endCol int) string {
	_, mid, _ := sliceDisplayColumnsParts(line, startCol, endCol)
	return mid
}

func sliceDisplayColumnsParts(line string, startCol, endCol int) (string, string, string) {
	if line == "" {
		return "", "", ""
	}
	if startCol < 0 {
		startCol = 0
	}
	if endCol < startCol {
		return line, "", ""
	}
	var before, selected, after strings.Builder
	col := 0
	for _, r := range line {
		rw := runewidth.RuneWidth(r)
		next := col + rw
		switch {
		case next <= startCol:
			before.WriteRune(r)
		case col > endCol:
			after.WriteRune(r)
		case col >= startCol && next-1 <= endCol:
			selected.WriteRune(r)
		default:
			selected.WriteRune(r)
		}
		col = next
	}
	return before.String(), selected.String(), after.String()
}
