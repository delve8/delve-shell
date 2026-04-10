package history

import (
	"strings"
	"testing"
)

func TestTruncateToolOutput_KeepHeadAndTail(t *testing.T) {
	head := strings.Repeat("A", 40*1024)
	tail := strings.Repeat("Z", 40*1024)
	in := head + strings.Repeat("M", 32*1024) + tail

	got := TruncateToolOutput(in)

	if len(got) > ToolOutputMaxBytes {
		t.Fatalf("len=%d want <= %d", len(got), ToolOutputMaxBytes)
	}
	if !strings.HasPrefix(got, head[:16*1024]) {
		t.Fatalf("missing preserved head prefix")
	}
	if !strings.HasSuffix(got, tail[len(tail)-16*1024:]) {
		t.Fatalf("missing preserved tail suffix")
	}
	if !strings.Contains(got, "[truncated, omitted ") {
		t.Fatalf("missing truncation notice")
	}
}

func TestTruncateToolOutput_UTF8Safe(t *testing.T) {
	in := strings.Repeat("中", ToolOutputMaxBytes) + strings.Repeat("尾", ToolOutputMaxBytes)
	got := TruncateToolOutput(in)
	if len(got) > ToolOutputMaxBytes {
		t.Fatalf("len=%d want <= %d", len(got), ToolOutputMaxBytes)
	}
	if !strings.HasPrefix(got, "中") {
		t.Fatalf("expected utf-8 safe head: %q", got[:minIntForTest(12, len(got))])
	}
	if !strings.HasSuffix(got, "尾") {
		t.Fatalf("expected utf-8 safe tail")
	}
}

func TestToolResultMessage_TruncatesLargeStdout(t *testing.T) {
	stdout := strings.Repeat("x", ToolOutputMaxBytes+1024)
	got := ToolResultMessage(stdout, "", 0, nil)
	if !strings.Contains(got, "stdout:\n") || !strings.Contains(got, "exit_code: 0") {
		t.Fatalf("unexpected shape: %q", got)
	}
	if !strings.Contains(got, "[truncated, omitted ") {
		t.Fatalf("expected truncation marker")
	}
}

func minIntForTest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
