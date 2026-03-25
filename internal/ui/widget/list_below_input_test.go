package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderLinesBelowInput_Empty(t *testing.T) {
	n := lipgloss.NewStyle()
	h := lipgloss.NewStyle().Bold(true)
	if s := RenderLinesBelowInput(" ", nil, n, h); s != "" {
		t.Fatalf("want empty, got %q", s)
	}
}

func TestRenderLinesBelowInput_PrefixAndHighlight(t *testing.T) {
	n := lipgloss.NewStyle()
	h := lipgloss.NewStyle().Bold(true)
	rows := []ListRow{
		{Text: "a", Highlight: false},
		{Text: "b", Highlight: true},
	}
	out := RenderLinesBelowInput("__", rows, n, h)
	if !strings.HasPrefix(out, "\n") {
		t.Fatalf("want leading newline: %q", out)
	}
	if !strings.Contains(out, "__a") || !strings.Contains(out, "__b") {
		t.Fatalf("missing prefixed text: %q", out)
	}
}
