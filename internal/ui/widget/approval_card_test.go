package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/hil/approvalview"
	hiltypes "delve-shell/internal/hil/types"
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
	lines := []approvalview.Line{
		{Kind: approvalview.LineHeader, Text: "H"},
		{Kind: approvalview.LineExec, Text: "cmd"},
	}
	out := RenderPendingApprovalLines(lines, s)
	if !strings.Contains(out, "H") || !strings.Contains(out, "cmd") || !strings.Contains(out, "\n") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestRenderPendingApprovalLines_autoApproveHL(t *testing.T) {
	plain := lipgloss.NewStyle()
	safe := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	risk := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	neutral := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	s := PendingCardStyles{
		Header:          plain,
		Exec:            plain,
		Suggest:         plain,
		RiskReadOnly:    plain,
		RiskLow:         plain,
		RiskHigh:        plain,
		ExecAutoSafe:    safe,
		ExecAutoRisk:    risk,
		ExecAutoNeutral: neutral,
	}
	text := "aa|bb"
	lines := []approvalview.Line{{
		Kind: approvalview.LineExec,
		Text: text,
		AutoApproveHL: []hiltypes.AutoApproveHighlightSpan{
			{Start: 0, End: 2, Kind: hiltypes.AutoApproveHighlightSafe},
			{Start: 2, End: 3, Kind: hiltypes.AutoApproveHighlightNeutral},
			{Start: 3, End: 5, Kind: hiltypes.AutoApproveHighlightRisk},
		},
	}}
	out := RenderPendingApprovalLines(lines, s)
	if out == "" || strings.Count(out, "aa") < 1 {
		t.Fatalf("unexpected: %q", out)
	}
}
