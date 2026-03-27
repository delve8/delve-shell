package ui

import (
	"strings"
	"testing"

	"delve-shell/internal/uivm"
)

// TUI (Bubble Tea) tests: do not run tea.Program; unit-test the Model by sending messages and asserting state/output.
// - Use nil or buffered chans to avoid blocking.
// - Call model.Update(tea.Msg) and assert on returned model state or model.View().
// - Config-dependent logic (e.g. getLang) falls back to defaults in tests; use inclusive asserts (e.g. accept both en and zh).

// TestView_FooterAlwaysShown asserts that View() always includes the footer status line (mode + status)
// and that total output lines never exceed Height so the footer stays visible when the terminal shows one screen.
func TestView_FooterAlwaysShown(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Height = 24
	m.layout.Width = 80
	m = m.WithTranscriptLines([]string{"hello"})
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > m.layout.Height {
		t.Fatalf("View() must not exceed Height: got %d lines, Height=%d", len(lines), m.layout.Height)
	}
	if strings.Contains(lines[0], "Auto-Run") || strings.Contains(lines[0], "自动执行") {
		t.Error("View() should not render the status line at the top anymore")
	}
	tailStart := len(lines) - 5
	if tailStart < 0 {
		tailStart = 0
	}
	footer := strings.Join(lines[tailStart:], "\n")
	if !strings.Contains(footer, "[IDLE]") && !strings.Contains(footer, "[空闲]") && !strings.Contains(footer, "[PROCESSING]") && !strings.Contains(footer, "[处理中]") {
		t.Error("View() should show status in the footer (e.g. [IDLE] or [空闲])")
	}
	if !strings.Contains(footer, "Auto-Run") && !strings.Contains(footer, "自动执行") {
		t.Error("View() should show Auto-Run label in the footer")
	}

	// Small height path: footer must still appear.
	m.layout.Height = 4
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "Auto-Run") && !strings.Contains(viewSmall, "自动执行") {
		t.Error("View() at small height should still show the footer with Auto-Run label")
	}

	// With Pending, footer shows [NEED APPROVAL] or [待确认]
	m.ChoiceCard.pending = &uivm.PendingApproval{Command: "ls"}
	m.layout.Height = 24
	m = m.syncChoiceViewport()
	viewPending := m.View()
	if !strings.Contains(viewPending, "[NEED APPROVAL]") && !strings.Contains(viewPending, "[待确认]") {
		t.Error("View() with Pending should show pending status in the footer")
	}

	// Critical: with choice mode (max 3 options) and a small Height, total lines must not exceed Height,
	// so the footer stays on screen when terminal displays one full screen.
	m2 := NewModel(nil, nil)
	m2.layout.Height = 12
	m2.layout.Width = 80
	m2.ChoiceCard.pendingSensitive = &uivm.PendingSensitive{Command: "cat /etc/shadow"}
	viewChoice := m2.View()
	choiceLines := strings.Split(viewChoice, "\n")
	if len(choiceLines) > m2.layout.Height {
		t.Errorf("View() in choice mode (3 options) must not exceed Height: got %d lines, Height=%d (footer would scroll off)", len(choiceLines), m2.layout.Height)
	}
	// Footer title must be in the visible area near the bottom, not at the top.
	visible := strings.Join(choiceLines[:min(len(choiceLines), m2.layout.Height)], "\n")
	if strings.Contains(choiceLines[0], "Auto-Run") || strings.Contains(choiceLines[0], "自动执行") {
		t.Error("footer should not appear in the first visible line")
	}
	if !strings.Contains(visible, "Auto-Run") && !strings.Contains(visible, "自动执行") {
		t.Error("footer (Auto-Run label) must appear in visible area")
	}
	if !strings.Contains(visible, "[NEED APPROVAL]") && !strings.Contains(visible, "[待确认]") {
		t.Error("footer (pending status) must appear in visible area")
	}
}

func TestMainTopPaddingLinesShrinksAsTranscriptPrints(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	m.layout.Height = 24
	bottom := renderSeparator(m.layout.Width) + "\n" + m.Input.View() + m.inputBelowBlock(m.getLang(), false) + m.footerLine()

	initialPad := m.normalModeTopPaddingLines(bottom)
	if initialPad <= 0 {
		t.Fatalf("expected positive initial top padding, got %d", initialPad)
	}

	m = m.AppendTranscriptLines("line1", "line2", "line3")
	m.printedMessages = len(m.messages)
	afterPrintPad := m.normalModeTopPaddingLines(bottom)
	if afterPrintPad >= initialPad {
		t.Fatalf("expected top padding to shrink after transcript prints, before=%d after=%d", initialPad, afterPrintPad)
	}
}

func TestMainTopPaddingLinesAccountsForTerminalWidth(t *testing.T) {
	wide := NewModel(nil, nil)
	wide.layout.Width = 80
	wide.layout.Height = 24
	wide = wide.AppendTranscriptLines(strings.Repeat("x", 60))
	wide.printedMessages = len(wide.messages)
	wideBottom := renderSeparator(wide.layout.Width) + "\n" + wide.Input.View() + wide.inputBelowBlock(wide.getLang(), false) + wide.footerLine()

	narrow := wide
	narrow.layout.Width = 20
	narrowBottom := renderSeparator(narrow.layout.Width) + "\n" + narrow.Input.View() + narrow.inputBelowBlock(narrow.getLang(), false) + narrow.footerLine()

	if narrow.normalModeTopPaddingLines(narrowBottom) >= wide.normalModeTopPaddingLines(wideBottom) {
		t.Fatalf("expected narrower terminal to leave less top padding, wide=%d narrow=%d", wide.normalModeTopPaddingLines(wideBottom), narrow.normalModeTopPaddingLines(narrowBottom))
	}
}

func TestTerminalWrappedRowsAccountsForSoftWrap(t *testing.T) {
	if got := terminalWrappedRows("", 10); got != 1 {
		t.Fatalf("empty message is one blank row (tea.Println), got %d", got)
	}
	text := "12345\nabcdef"
	if got := terminalWrappedRows(text, 10); got != 2 {
		t.Fatalf("expected 2 display lines at width 10, got %d", got)
	}
	if got := terminalWrappedRows(text, 3); got != 4 {
		t.Fatalf("expected 4 display lines at width 3, got %d", got)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
