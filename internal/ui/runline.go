package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// IsRunTranscriptExecLine reports whether s is a compact execute_command transcript line
// ("Run (...): ..."), after stripping ANSI. Used to avoid textwrap splitting it into multiple rows.
func IsRunTranscriptExecLine(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(ansi.Strip(s)), "Run (")
}

// RunTranscriptLineMaxWidth caps display width for post-exec "Run (...): <cmd>" transcript lines only.
// Pending approval cards do not use [FormatRunTranscriptLine] — they show the full command (approvalview + widget).
const RunTranscriptLineMaxWidth = 100

// FormatRunTranscriptLine builds one logical transcript line for after execution: prefix + trimmed cmd.
// Long lines are truncated with a "...." tail so the scrollback stays compact; the approval card still
// shows the full command via a separate path. Used by presenter, history replay, and suggested-dismiss line.
func FormatRunTranscriptLine(prefix, cmd string) string {
	cmd = compactRunTranscriptCommand(cmd)
	s := prefix + cmd
	if ansi.StringWidth(s) <= RunTranscriptLineMaxWidth {
		return s
	}
	return ansi.Truncate(s, RunTranscriptLineMaxWidth, "....")
}

// FormatRunTranscriptLineFull is like [FormatRunTranscriptLine] but never truncates the command;
// use for read-only overlays (e.g. /history preview) where the full command must be visible.
func FormatRunTranscriptLineFull(prefix, cmd string) string {
	cmd = strings.TrimSpace(strings.ReplaceAll(cmd, "\r", ""))
	if strings.Contains(cmd, "\n") {
		return formatMultilineRunTranscript(prefix, cmd, 0)
	}
	return prefix + cmd
}

// RunTranscriptDisplayMaxCells is the display cap for a printed Run line: never wider than
// [RunTranscriptLineMaxWidth], and never wider than the terminal content width so tea.Println does not
// soft-wrap into extra rows (which desyncs [Model.printedTranscriptLineCount] and can merge the next
// View() row with input placeholder text).
func RunTranscriptDisplayMaxCells(termWidth int) int {
	if termWidth < 1 {
		termWidth = 1
	}
	if termWidth > RunTranscriptLineMaxWidth {
		return RunTranscriptLineMaxWidth
	}
	return termWidth
}

// ClampRunTranscriptPlain re-truncates a plain "Run (…): cmd" line to maxCells display width.
// maxCells should be [RunTranscriptDisplayMaxCells] for the current terminal.
func ClampRunTranscriptPlain(plain string, maxCells int) string {
	if maxCells < 1 {
		maxCells = 1
	}
	plain = compactRunTranscriptPlain(plain)
	if !strings.HasPrefix(plain, "Run (") {
		if ansi.StringWidth(plain) <= maxCells {
			return plain
		}
		return ansi.Truncate(plain, maxCells, "....")
	}
	idx := strings.Index(plain, "): ")
	if idx < 0 {
		if ansi.StringWidth(plain) <= maxCells {
			return plain
		}
		return ansi.Truncate(plain, maxCells, "....")
	}
	s := plain[:idx+3] + strings.TrimSpace(plain[idx+3:])
	if ansi.StringWidth(s) <= maxCells {
		return s
	}
	return ansi.Truncate(s, maxCells, "....")
}

func formatMultilineRunTranscript(prefix, cmd string, maxCells int) string {
	lines := strings.Split(cmd, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	contPrefix := strings.Repeat(" ", ansi.StringWidth(prefix))
	rendered := make([]string, 0, len(lines))
	for i, line := range lines {
		out := line
		if i == 0 {
			out = prefix + out
		} else {
			out = contPrefix + out
		}
		if maxCells > 0 && ansi.StringWidth(out) > maxCells {
			out = ansi.Truncate(out, maxCells, "....")
		}
		rendered = append(rendered, out)
	}
	return strings.Join(rendered, "\n")
}

func compactRunTranscriptCommand(cmd string) string {
	cmd = strings.TrimSpace(strings.ReplaceAll(cmd, "\r", ""))
	if cmd == "" {
		return ""
	}
	return strings.Join(strings.Fields(cmd), " ")
}

func compactRunTranscriptPlain(plain string) string {
	plain = strings.TrimSpace(strings.ReplaceAll(plain, "\r", ""))
	if plain == "" {
		return ""
	}
	idx := strings.Index(plain, "): ")
	if idx < 0 {
		return strings.Join(strings.Fields(plain), " ")
	}
	return plain[:idx+3] + compactRunTranscriptCommand(plain[idx+3:])
}
