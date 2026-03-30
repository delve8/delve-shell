package history

import (
	"strings"
	"testing"
)

func TestFormatSessionSnippetForDisplay_newlinesAndTruncate(t *testing.T) {
	t.Parallel()
	got := FormatSessionSnippetForDisplay("line1\nline2", 80)
	if want := `line1\nline2`; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	long := strings.Repeat("あ", 40) // 40 runes
	got = FormatSessionSnippetForDisplay(long+"\nx", 20)
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
	if gotR := []rune(got); len(gotR) != 20 {
		t.Fatalf("expected total display width 20 runes, got %d (%q)", len(gotR), got)
	}
}
