package widget

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderFooterBar_fullWidthUsesSpacing(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(80, FooterBarParts{
		Remote:       "Local",
		AutoRunFull:  "Auto-Run: List Only",
		AutoRunShort: "AR:list",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Contains(out, "·") || !strings.Contains(out, "Local") || !strings.Contains(out, "[IDLE]") {
		t.Fatalf("unexpected: %q", out)
	}
	if !strings.Contains(out, "   Auto-Run") {
		t.Fatalf("expected wider spacing between segments in %q", out)
	}
}

func TestRenderFooterBar_truncatesRemoteMiddle(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(44, FooterBarParts{
		Remote:       "Remote alpha-bravo-charlie-XYZ1",
		AutoRunFull:  "Auto-Run: List Only",
		AutoRunShort: "AR:list",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if !strings.Contains(out, "…") {
		t.Fatalf("expected middle ellipsis in %q", out)
	}
	if !strings.Contains(out, "Remote") || !strings.Contains(out, "XYZ1") {
		t.Fatalf("expected both ends of remote label in %q", out)
	}
}

func TestRenderFooterBar_autoRunWidthIsStable(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	on := RenderFooterBar(80, FooterBarParts{
		Remote:              "Local",
		AutoRunFull:         "Auto-Run: List Only",
		AutoRunShort:        "AR:list",
		AutoRunReserveWidth: 19,
		Status:              "[IDLE]",
	}, TitleBarStatusIdle, s)
	off := RenderFooterBar(80, FooterBarParts{
		Remote:              "Local",
		AutoRunFull:         "Auto-Run: None",
		AutoRunShort:        "AR:off",
		AutoRunReserveWidth: 19,
		Status:              "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Index(on, "Local") != strings.Index(off, "Local") {
		t.Fatalf("expected remote segment to stay fixed: on=%q off=%q", on, off)
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
