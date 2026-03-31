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

func TestRenderFooterBar_omitAutoUsesSingleSep(t *testing.T) {
	plain := lipgloss.NewStyle()
	s := TitleLineStyles{
		Base:       plain,
		StatusIdle: plain,
	}
	out := RenderFooterBar(80, FooterBarParts{
		Remote:              "Local",
		AutoRunReserveWidth: 0,
		Status:              "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Contains(out, "Auto-Run") {
		t.Fatalf("omit auto should not render middle segment: %q", out)
	}
	if !strings.Contains(out, "Local") || !strings.Contains(out, "[IDLE]") {
		t.Fatalf("expected status and remote: %q", out)
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
	a := RenderFooterBar(80, FooterBarParts{
		Remote:              "Local",
		AutoRunFull:         "Auto-Run: List Only",
		AutoRunShort:        "AR:list",
		AutoRunReserveWidth: 19,
		Status:              "[IDLE]",
	}, TitleBarStatusIdle, s)
	b := RenderFooterBar(80, FooterBarParts{
		Remote:              "Local",
		AutoRunFull:         "Auto-Run: List Only",
		AutoRunShort:        "AR:list",
		AutoRunReserveWidth: 19,
		Status:              "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Index(a, "Local") != strings.Index(b, "Local") {
		t.Fatalf("expected remote segment to stay fixed: a=%q b=%q", a, b)
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
		AutoRunFull:  "Auto-Run: List Only",
		AutoRunShort: "AR:list",
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
		AutoRunFull:  "Auto-Run: List Only",
		AutoRunShort: "AR:list",
		Status:       "[IDLE]",
	}, TitleBarStatusIdle, s)
	if strings.Contains(out, "Auto-Run: List Only") {
		t.Fatalf("expected short auto-run label to stay active while remote shrinks in %q", out)
	}
	if !strings.Contains(out, "AR:list") {
		t.Fatalf("expected short auto-run label in %q", out)
	}
}

func TestRenderFooterBar_statusOtherUsesBaseForStatusAndAuto(t *testing.T) {
	mark := lipgloss.NewStyle().Bold(true)
	s := TitleLineStyles{Base: mark}
	out := RenderFooterBar(100, FooterBarParts{
		Status:              "y",
		AutoRunFull:         "x",
		AutoRunReserveWidth: 1,
	}, TitleBarStatusOther, s)
	if out == "" || !strings.Contains(out, "x") || !strings.Contains(out, "y") {
		t.Fatalf("unexpected: %q", out)
	}
}
