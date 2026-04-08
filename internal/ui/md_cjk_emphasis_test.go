package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRelaxMarkdownStrongAdjacentCJK_userExample(t *testing.T) {
	const in = `一个**"异常 Pod + 镜像拉取 secret"**专项排查`
	got := relaxMarkdownStrongAdjacentCJK(in)
	if got == in {
		t.Fatalf("expected spaces inserted for CJK/** adjacency, got unchanged: %q", got)
	}
	lines := renderAILineTranscript(got, 100)
	joined := strings.Join(lines, "\n")
	plain := ansi.Strip(joined)
	if strings.Contains(plain, "**") {
		t.Fatalf("expected strong parsed (no literal ** in plain text): %q", plain)
	}
}

func TestRelaxMarkdownStrongAdjacentCJK_listItemOpening(t *testing.T) {
	const in = `1. **按 namespace 汇总 Pod 健康状况**`
	got := relaxMarkdownStrongAdjacentCJK(in)
	if strings.Contains(got, "** ") {
		t.Fatalf("opening ** must not be followed by space (invalid CommonMark): %q", got)
	}
	if got != in {
		t.Fatalf("expected unchanged for list + **CJK open, got %q", got)
	}
}
