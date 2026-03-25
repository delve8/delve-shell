package ui

import (
	"strings"
	"testing"

	"delve-shell/internal/agent"
)

// TUI (Bubble Tea) tests: do not run tea.Program; unit-test the Model by sending messages and asserting state/output.
// - Use nil or buffered chans to avoid blocking.
// - Call model.Update(tea.Msg) and assert on returned model state or model.View() / model.buildContent().
// - Config-dependent logic (e.g. getLang) falls back to defaults in tests; use inclusive asserts (e.g. accept both en and zh).

// TestView_HeaderAlwaysShown asserts that View() always includes the header (mode + status) and that
// total output lines never exceed Height so the header stays visible when the terminal shows one screen.
func TestView_HeaderAlwaysShown(t *testing.T) {
	m := NewModel(nil, false)
	m.Layout.Height = 24
	m.Layout.Width = 80
	view := m.View()
	// Header contains Auto-run label and a status in brackets
	if !strings.Contains(view, "[IDLE]") && !strings.Contains(view, "[空闲]") && !strings.Contains(view, "[PROCESSING]") && !strings.Contains(view, "[处理中]") {
		t.Error("View() should show status in header (e.g. [IDLE] or [空闲])")
	}
	if !strings.Contains(view, "Auto-Run") && !strings.Contains(view, "自动执行") {
		t.Error("View() should show Auto-Run label in header")
	}

	// Small height path: header must still appear first
	m.Layout.Height = 4
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "Auto-Run") && !strings.Contains(viewSmall, "自动执行") {
		t.Error("View() at small height should still show header with Auto-Run label")
	}

	// With Pending, header shows [NEED APPROVAL] or [待确认]
	ch := make(chan agent.ApprovalResponse, 1)
	m.Approval.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.Layout.Height = 24
	viewPending := m.View()
	if !strings.Contains(viewPending, "[NEED APPROVAL]") && !strings.Contains(viewPending, "[待确认]") {
		t.Error("View() with Pending should show pending status in header")
	}

	// Critical: with choice mode (max 3 options) and a small Height, total lines must not exceed Height,
	// so the header (first 2 lines) stays on screen when terminal displays one full screen.
	m2 := NewModel(nil, false)
	m2.Layout.Height = 12
	m2.Layout.Width = 80
	m2.Approval.PendingSensitive = &agent.SensitiveConfirmationRequest{Command: "cat /etc/shadow", ResponseCh: make(chan agent.SensitiveChoice, 1)}
	viewChoice := m2.View()
	lines := strings.Split(viewChoice, "\n")
	if len(lines) > m2.Layout.Height {
		t.Errorf("View() in choice mode (3 options) must not exceed Height: got %d lines, Height=%d (header would scroll off)", len(lines), m2.Layout.Height)
	}
	// First line must be the header title (Auto-Run + status)
	visible := strings.Join(lines[:min(len(lines), m2.Layout.Height)], "\n")
	if !strings.Contains(visible, "Auto-Run") && !strings.Contains(visible, "自动执行") {
		t.Error("header (Auto-Run label) must appear in visible area")
	}
	if !strings.Contains(visible, "[NEED APPROVAL]") && !strings.Contains(visible, "[待确认]") {
		t.Error("header (pending status) must appear in visible area")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
