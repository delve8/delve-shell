package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderTitleLine_idle(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderTitleLine("A | ", "[IDLE]", TitleBarStatusIdle, s)
	if !strings.Contains(out, "A | ") || !strings.Contains(out, "[IDLE]") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestRenderTitleLine_otherUsesBaseOnly(t *testing.T) {
	mark := lipgloss.NewStyle().Bold(true)
	s := TitleLineStyles{Base: mark}
	out := RenderTitleLine("x", "y", TitleBarStatusOther, s)
	if out == "" || !strings.Contains(out, "x") || !strings.Contains(out, "y") {
		t.Fatalf("unexpected: %q", out)
	}
}
