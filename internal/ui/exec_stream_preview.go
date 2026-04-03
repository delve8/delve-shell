package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/i18n"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/uivm"
)

const (
	execStreamPreviewMaxLines = 3
	// execStreamPreviewHeaderRows: one dim label above the rolling lines.
	execStreamPreviewHeaderRows = 1
)

func (m *Model) execStreamPreviewReserveRows() int {
	if !m.Interaction.execStreamWindowOpen {
		return 0
	}
	return execStreamPreviewHeaderRows + execStreamPreviewMaxLines
}

// renderExecStreamPreviewBlock returns a block (with trailing newline when non-empty) above the main separator.
func (m *Model) renderExecStreamPreviewBlock() string {
	if !m.Interaction.execStreamWindowOpen {
		return ""
	}
	w := m.contentWidth()
	if w < 1 {
		w = 1
	}
	header := suggestStyle.Render(textwrap.WrapString(i18n.T(i18n.KeyExecStreamPreviewHeader), w))
	lines := m.lastExecStreamPreviewLogicalLines()
	rows := make([]string, 0, execStreamPreviewMaxLines)
	for i := 0; i < execStreamPreviewMaxLines; i++ {
		if i < len(lines) {
			rows = append(rows, m.renderExecStreamPreviewLine(lines[i], w))
			continue
		}
		rows = append(rows, suggestStyle.Render(strings.Repeat("·", minInt(3, w))))
	}
	return header + "\n" + strings.Join(rows, "\n") + "\n"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Model) lastExecStreamPreviewLogicalLines() []execStreamSeg {
	buf := m.Interaction.execStreamBuffer
	if len(buf) == 0 {
		return nil
	}
	start := len(buf) - execStreamPreviewMaxLines
	if start < 0 {
		start = 0
	}
	out := make([]execStreamSeg, len(buf)-start)
	copy(out, buf[start:])
	return out
}

func (m *Model) renderExecStreamPreviewLine(seg execStreamSeg, w int) string {
	text := seg.text
	if seg.stderr {
		text = "stderr: " + text
	}
	plain := ansi.Strip(strings.ReplaceAll(text, "\r", ""))
	single := strings.Join(strings.Fields(strings.ReplaceAll(plain, "\n", " ")), " ")
	if single == "" {
		single = " "
	}
	wrapped := textwrap.WrapString(single, w)
	parts := strings.Split(wrapped, "\n")
	last := parts[len(parts)-1]
	line := resultStyle.Render(last)
	if w > 0 && ansi.StringWidth(line) > w {
		line = ansi.Truncate(line, w, "")
	}
	return line
}

func (m *Model) handleExecStreamWindowOpenMsg() (*Model, tea.Cmd) {
	m.Interaction.execStreamWindowOpen = true
	return m, nil
}

func (m *Model) handleExecStreamPreviewMsg(msg ExecStreamPreviewMsg) (*Model, tea.Cmd) {
	t := strings.TrimRight(msg.Line, "\r\n")
	if t == "" {
		return m, nil
	}
	m.Interaction.execStreamBuffer = append(m.Interaction.execStreamBuffer, execStreamSeg{text: t, stderr: msg.Stderr})
	return m, nil
}

func (m *Model) handleExecStreamFlushMsg(msg ExecStreamFlushMsg) (*Model, tea.Cmd) {
	m.Interaction.execStreamWindowOpen = false
	var lines []uivm.Line
	for _, seg := range m.Interaction.execStreamBuffer {
		text := seg.text
		if seg.stderr {
			text = "stderr: " + text
		}
		lines = append(lines, uivm.Line{Kind: uivm.LineResult, Text: text})
	}
	m.Interaction.execStreamBuffer = nil
	if msg.Sensitive {
		lines = append(lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Result contains sensitive data."})
	}
	if msg.Tail != "" {
		lines = append(lines, uivm.Line{Kind: uivm.LineResult, Text: msg.Tail})
	}
	lines = append(lines, uivm.Line{Kind: uivm.LineBlank})
	rendered := m.renderTranscriptLines(lines)
	m.AppendTranscriptLines(rendered...)
	return m, m.printTranscriptCmd(false)
}
