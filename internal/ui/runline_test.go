package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFormatRunTranscriptLine_shortUnchanged(t *testing.T) {
	got := FormatRunTranscriptLine("Run (direct): ", "ls -la")
	want := "Run (direct): ls -la"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsRunTranscriptExecLine(t *testing.T) {
	if !IsRunTranscriptExecLine("Run (direct): ls") {
		t.Fatal("plain")
	}
	styled := "\x1b[31mRun (approved): ls\x1b[0m"
	if !IsRunTranscriptExecLine(styled) {
		t.Fatal("styled")
	}
	if IsRunTranscriptExecLine("kubectl get pods") {
		t.Fatal("should be false for non-run line")
	}
}

func TestFormatRunTranscriptLineFull_neverTruncates(t *testing.T) {
	cmd := strings.Repeat("x", RunTranscriptLineMaxWidth+50)
	got := FormatRunTranscriptLineFull("Run (approved): ", cmd)
	wantLen := ansi.StringWidth("Run (approved): ") + len(cmd)
	if ansi.StringWidth(got) != wantLen {
		t.Fatalf("full line width %d, want %d", ansi.StringWidth(got), wantLen)
	}
}

func TestFormatRunTranscriptLine_truncatesLongLine(t *testing.T) {
	cmd := strings.Repeat("x", RunTranscriptLineMaxWidth+50)
	got := FormatRunTranscriptLine("Run (checks passed): ", cmd)
	if ansi.StringWidth(got) > RunTranscriptLineMaxWidth {
		t.Fatalf("width %d > max %d", ansi.StringWidth(got), RunTranscriptLineMaxWidth)
	}
	if !strings.HasSuffix(got, "....") {
		t.Fatalf("expected .... tail, got %q", got)
	}
}

func TestFormatRunTranscriptLine_multilineCompactedToOneLine(t *testing.T) {
	prefix := "Run (approved): "
	got := FormatRunTranscriptLine(prefix, "kubectl get nodes \\\n  -o wide\nkubectl get pods -A")
	want := prefix + "kubectl get nodes \\ -o wide kubectl get pods -A"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatRunTranscriptLineFull_multilinePreserved(t *testing.T) {
	prefix := "Run (approved): "
	indent := strings.Repeat(" ", ansi.StringWidth(prefix))
	got := FormatRunTranscriptLineFull(prefix, "echo a\n  echo b")
	want := prefix + "echo a\n" + indent + "  echo b"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRunTranscriptDisplayMaxCells(t *testing.T) {
	if g := RunTranscriptDisplayMaxCells(80); g != 80 {
		t.Fatalf("got %d want 80", g)
	}
	if g := RunTranscriptDisplayMaxCells(200); g != RunTranscriptLineMaxWidth {
		t.Fatalf("wide term: got %d want cap %d", g, RunTranscriptLineMaxWidth)
	}
	if g := RunTranscriptDisplayMaxCells(0); g != 1 {
		t.Fatalf("non-positive: got %d want 1", g)
	}
}

func TestClampRunTranscriptPlain_narrowerThanPresenterCap(t *testing.T) {
	long := FormatRunTranscriptLine("Run (approved): ", strings.Repeat("a", RunTranscriptLineMaxWidth+30))
	if ansi.StringWidth(long) > RunTranscriptLineMaxWidth {
		t.Fatalf("setup: presenter cap line width %d", ansi.StringWidth(long))
	}
	const termW = 48
	clamped := ClampRunTranscriptPlain(long, RunTranscriptDisplayMaxCells(termW))
	if w := ansi.StringWidth(clamped); w > termW {
		t.Fatalf("width %d > term %d: %q", w, termW, clamped)
	}
	if !strings.HasSuffix(clamped, "....") {
		t.Fatalf("expected .... tail: %q", clamped)
	}
}

func TestClampRunTranscriptPlain_multilineCompactsThenClamps(t *testing.T) {
	plain := "Run (approved): echo hello \\\n  world\nkubectl get pods -A"
	clamped := ClampRunTranscriptPlain(plain, 40)
	if strings.Contains(clamped, "\n") {
		t.Fatalf("expected single line, got %q", clamped)
	}
	if w := ansi.StringWidth(clamped); w > 40 {
		t.Fatalf("width %d > 40: %q", w, clamped)
	}
	if !strings.HasPrefix(clamped, "Run (approved): echo hello \\ world") {
		t.Fatalf("unexpected compacted line: %q", clamped)
	}
}
