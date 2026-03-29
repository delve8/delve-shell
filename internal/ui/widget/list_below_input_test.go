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

func TestRenderLinesBelowInput_PreRenderedSkipsNormalStyle(t *testing.T) {
	n := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	h := lipgloss.NewStyle()
	raw := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("__pre")
	out := RenderLinesBelowInput("XX", []ListRow{{Text: raw, PreRendered: true}}, n, h)
	if strings.Contains(out, "\x1b[31mXX") { // red from n applied to prefix+text
		t.Fatalf("pre-rendered row should not wrap with normal style: %q", out)
	}
	if !strings.Contains(out, raw) {
		t.Fatalf("want pre-rendered segment in output: %q", out)
	}
}

func TestRenderFixedLinesBelowInput_PadsBlanks(t *testing.T) {
	n := lipgloss.NewStyle()
	h := lipgloss.NewStyle().Bold(true)
	out := RenderFixedLinesBelowInput("__", []ListRow{{Text: "a"}}, 3, n, h)
	if strings.Count(out, "\n") != 4 {
		t.Fatalf("expected 4 newline chars for 3 reserved rows, got %q", out)
	}
	if !strings.Contains(out, "__a") {
		t.Fatalf("missing rendered row: %q", out)
	}
}
