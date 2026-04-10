package history

import (
	"fmt"
	"strconv"
	"unicode/utf8"
)

const (
	// ToolOutputMaxBytes caps one stdout/stderr payload kept in history or returned to the model.
	ToolOutputMaxBytes = 64 * 1024
)

// TruncateToolOutput keeps the start and end of a long output while removing the middle.
// The returned string is UTF-8 safe and no longer than [ToolOutputMaxBytes] bytes.
func TruncateToolOutput(s string) string {
	return TruncateMiddleText(s, ToolOutputMaxBytes)
}

// RedactAndTruncateToolOutput applies heuristic redaction and then caps the result.
func RedactAndTruncateToolOutput(s string) string {
	return TruncateToolOutput(RedactText(s))
}

// ToolResultMessage builds the normal execute_command / run_skill return shape with truncated stdout/stderr.
func ToolResultMessage(stdout, stderr string, exitCode int, execErr error) string {
	stdout = TruncateToolOutput(stdout)
	stderr = TruncateToolOutput(stderr)
	msg := "stdout:\n" + stdout
	if stderr != "" {
		msg += "\nstderr:\n" + stderr
	}
	msg += "\nexit_code: " + strconv.Itoa(exitCode)
	if execErr != nil && exitCode == 0 {
		msg += "\nerror: " + TruncateToolOutput(execErr.Error())
	}
	return msg
}

// TruncateMiddleText keeps the head and tail of s and inserts a truncation notice in the middle
// when s exceeds maxBytes. The returned string is UTF-8 safe and no longer than maxBytes bytes.
func TruncateMiddleText(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	notice := fmt.Sprintf("\n...[truncated, omitted %d bytes]...\n", len(s)-maxBytes)
	if len(notice) >= maxBytes {
		return utf8SafePrefix(notice, maxBytes)
	}
	keep := maxBytes - len(notice)
	headBudget := keep / 2
	tailBudget := keep - headBudget
	head := utf8SafePrefix(s, headBudget)
	tail := utf8SafeSuffix(s, tailBudget)
	omitted := len(s) - len(head) - len(tail)
	if omitted < 0 {
		omitted = 0
	}
	notice = fmt.Sprintf("\n...[truncated, omitted %d bytes]...\n", omitted)
	if len(notice) > maxBytes {
		notice = utf8SafePrefix(notice, maxBytes)
	}
	keep = maxBytes - len(notice)
	if keep < 0 {
		keep = 0
	}
	headBudget = keep / 2
	tailBudget = keep - headBudget
	head = utf8SafePrefix(s, headBudget)
	tail = utf8SafeSuffix(s, tailBudget)
	out := head + notice + tail
	if len(out) <= maxBytes {
		return out
	}
	return utf8SafePrefix(out, maxBytes)
}

func utf8SafePrefix(s string, maxBytes int) string {
	if maxBytes <= 0 || s == "" {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(s[:cut]) {
		cut--
	}
	return s[:cut]
}

func utf8SafeSuffix(s string, maxBytes int) string {
	if maxBytes <= 0 || s == "" {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	start := len(s) - maxBytes
	for start < len(s) && !utf8.ValidString(s[start:]) {
		start++
	}
	return s[start:]
}
