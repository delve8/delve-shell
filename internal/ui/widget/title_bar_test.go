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
	if !strings.Contains(out, "        Auto-Run") {
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
	if !strings.Contains(out, "…") || !strings.Contains(out, "1") {
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

func TestRenderFooterBar_shrinksSpacingBeforeRemote(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(38, FooterBarParts{
		Remote:       "Local",
		AutoRunFull:  "Auto-Run: None",
		AutoRunShort: "AR:off",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if !strings.Contains(out, "  Auto-Run") {
		t.Fatalf("expected footer to retain at least a 2-space separator in %q", out)
	}
	if !strings.Contains(out, "Local") {
		t.Fatalf("expected remote text to survive after spacing shrink in %q", out)
	}
}

func TestRenderFooterBar_usesShortAutoRunBeforeTruncatingRemote(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(28, FooterBarParts{
		Remote:       "Local",
		AutoRunFull:  "Auto-Run: List Only",
		AutoRunShort: "AR:list",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if !strings.Contains(out, "AR:list") {
		t.Fatalf("expected short auto-run label in %q", out)
	}
	if !strings.Contains(out, "Local") {
		t.Fatalf("expected remote to remain visible after auto-run shortens in %q", out)
	}
}

func TestRenderFooterBar_keepsShortAutoRunWhileRemoteShrinks(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(24, FooterBarParts{
		Remote:       "Remote-XYZ1",
		AutoRunFull:  "Auto-Run: None",
		AutoRunShort: "AR:off",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Contains(out, "Auto-Run: None") {
		t.Fatalf("expected short auto-run label to stay active while remote shrinks in %q", out)
	}
	if !strings.Contains(out, "AR:off") {
		t.Fatalf("expected short auto-run label in %q", out)
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
