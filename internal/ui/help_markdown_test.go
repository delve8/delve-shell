package ui

import (
	"strings"
	"testing"
)

func TestRenderHelpMarkdown_nonEmpty(t *testing.T) {
	out := RenderHelpMarkdown("# Title\n\nHello **world**.", 60)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected rendered output")
	}
	if !strings.Contains(out, "Title") || !strings.Contains(out, "world") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, uiVersionText()) {
		t.Fatalf("expected rendered help to include version %q, got %q", uiVersionText(), out)
	}
}
