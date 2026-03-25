package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/uiflow/choicecard"
)

func TestRenderPendingApprovalLines_empty(t *testing.T) {
	s := PendingCardStyles{
		Header: lipgloss.NewStyle(),
	}
	if out := RenderPendingApprovalLines(nil, s); out != "" {
		t.Fatalf("want empty, got %q", out)
	}
}

func TestRenderPendingApprovalLines_joinsKinds(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := PendingCardStyles{
		Header:       plain,
		Exec:         plain,
		Suggest:      plain,
		RiskReadOnly: plain,
		RiskLow:      plain,
		RiskHigh:     plain,
	}
	lines := []choicecard.Line{
		{Kind: choicecard.LineHeader, Text: "H"},
		{Kind: choicecard.LineExec, Text: "cmd"},
	}
	out := RenderPendingApprovalLines(lines, s)
	if !strings.Contains(out, "H") || !strings.Contains(out, "cmd") || !strings.Contains(out, "\n") {
		t.Fatalf("unexpected: %q", out)
	}
}
