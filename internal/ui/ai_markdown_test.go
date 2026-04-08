package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderAILineTranscript_narrowUsesPlainWrap(t *testing.T) {
	md := "# Title\n\nBody."
	// innerW = 20 < minAIMarkdownInnerWidth -> plain path, no ANSI styling
	lines := renderAILineTranscript(md, 20)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "\x1b[") {
		t.Fatalf("expected plain wrap without ANSI, got: %q", joined)
	}
	if !strings.Contains(joined, "#") {
		t.Fatalf("expected raw markdown markers in fallback: %q", joined)
	}
}

func TestRenderAILineTranscript_markdownBold(t *testing.T) {
	md := "Use **bold** text."
	lines := renderAILineTranscript(md, 80)
	joined := strings.Join(lines, "\n")
	plain := ansi.Strip(joined)
	if !strings.Contains(plain, "bold") {
		t.Fatalf("expected rendered text to contain bold: %q", plain)
	}
	if strings.Contains(plain, "**") {
		t.Fatalf("expected markdown asterisks stripped in output: %q", plain)
	}
}

func TestRenderAILineTranscript_emptyBody(t *testing.T) {
	lines := renderAILineTranscript("", 80)
	if len(lines) != 0 {
		t.Fatalf("expected no lines for empty body, got %#v", lines)
	}
}
