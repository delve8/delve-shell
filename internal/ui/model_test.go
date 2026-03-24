package ui

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
)

// TUI (Bubble Tea) tests: do not run tea.Program; unit-test the Model by sending messages and asserting state/output.
// - Use nil or buffered chans to avoid blocking.
// - Call model.Update(tea.Msg) and assert on returned model state or model.View() / model.buildContent().
// - Config-dependent logic (e.g. getLang) falls back to defaults in tests; use inclusive asserts (e.g. accept both en and zh).

func TestApprovalCard_ShowsCommandReasonAndRisk(t *testing.T) {
	// do not run tea.NewProgram().Run(); just build Model and set Pending
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	ch := make(chan agent.ApprovalResponse, 1)
	m.Approval.Pending = &agent.ApprovalRequest{
		Command:    "kubectl get pods",
		Reason:     "List pods to check status",
		RiskLevel:  "read_only",
		ResponseCh: ch,
	}
	content := m.buildContent()

	if !strings.Contains(content, "kubectl get pods") {
		t.Error("approval card should show command")
	}
	if !strings.Contains(content, "List pods to check status") {
		t.Error("approval card should show AI reason")
	}
	// risk label varies by language; at least one must appear
	if !strings.Contains(content, "READ-ONLY") && !strings.Contains(content, "只读") {
		t.Error("approval card should show read_only risk label (en or zh)")
	}
}

func TestApprovalCard_HighRiskLabel(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Approval.Pending = &agent.ApprovalRequest{
		Command:    "rm -rf /tmp/foo",
		RiskLevel:  "high",
		ResponseCh: make(chan agent.ApprovalResponse, 1),
	}
	content := m.buildContent()

	if !strings.Contains(content, "rm -rf /tmp/foo") {
		t.Error("approval card should show command")
	}
	if !strings.Contains(content, "HIGH-RISK") && !strings.Contains(content, "高风险") {
		t.Error("approval card should show high risk label (en or zh)")
	}
}

func TestApprovalCard_Approve1ClearsPending(t *testing.T) {
	ch := make(chan agent.ApprovalResponse, 1)
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Approval.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	// simulate user pressing 1 (approve)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m2 := next.(Model)
	if m2.Approval.Pending != nil {
		t.Error("pending should be cleared after 1")
	}
	select {
	case v := <-ch:
		if !v.Approved {
			t.Error("channel should receive Approved true for approve")
		}
	default:
		t.Error("channel should receive response")
	}
}

func TestApprovalCard_Approve2ClearsPendingAndSendsFalse(t *testing.T) {
	ch := make(chan agent.ApprovalResponse, 1)
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Approval.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m2 := next.(Model)
	if m2.Approval.Pending != nil {
		t.Error("pending should be cleared after 2")
	}
	select {
	case v := <-ch:
		if v.Approved {
			t.Error("channel should receive Approved false for reject")
		}
	default:
		t.Error("channel should receive response")
	}
}

// TestView_HeaderAlwaysShown asserts that View() always includes the header (mode + status) and that
// total output lines never exceed Height so the header stays visible when the terminal shows one screen.
func TestView_HeaderAlwaysShown(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
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
	m2 := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func() bool { return true }, nil, "", false)
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

// TestChoice_EnterSelectsCurrentOption asserts that pressing Enter in choice mode selects the current (highlighted) option.
func TestChoice_EnterSelectsCurrentOption(t *testing.T) {
	// Approval: ChoiceIndex 0 = approve, Enter should send Approved true
	ch := make(chan agent.ApprovalResponse, 1)
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Approval.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.Interaction.ChoiceIndex = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if m2.Approval.Pending != nil {
		t.Error("pending should be cleared after Enter")
	}
	select {
	case v := <-ch:
		if !v.Approved {
			t.Error("Enter with ChoiceIndex 0 should approve (Approved true)")
		}
	default:
		t.Error("channel should receive response")
	}

	// Approval: ChoiceIndex 1 = reject, Enter should send Approved false
	ch2 := make(chan agent.ApprovalResponse, 1)
	m3 := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m3.Approval.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch2}
	m3.Interaction.ChoiceIndex = 1

	next2, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := next2.(Model)
	if m4.Approval.Pending != nil {
		t.Error("pending should be cleared after Enter on option 2")
	}
	select {
	case v := <-ch2:
		if v.Approved {
			t.Error("Enter with ChoiceIndex 1 should reject (Approved false)")
		}
	default:
		t.Error("channel should receive response")
	}
}

// TestSessionSwitchedMsg_setsCurrentPathAndShowsSwitchedAtBottom asserts that after SessionSwitchedMsg,
// CurrentSessionPath is set and the "Switched to session" line is present (at end when there is history).
func TestSessionSwitchedMsg_setsCurrentPathAndShowsSwitchedAtBottom(t *testing.T) {
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func() bool { return true }, nil, "", false)

	// Path empty: switched hint then blank line, CurrentSessionPath ""
	next, _ := m.Update(SessionSwitchedMsg{Path: ""})
	m2 := next.(Model)
	if m2.Context.CurrentSessionPath != "" {
		t.Errorf("CurrentSessionPath should be empty when Path is empty, got %q", m2.Context.CurrentSessionPath)
	}
	if len(m2.Messages) < 1 {
		t.Errorf("expected at least 1 message when Path empty, got %d", len(m2.Messages))
	}
	if !strings.Contains(m2.Messages[0], "Switched") && !strings.Contains(m2.Messages[0], "切换") {
		t.Errorf("message should contain Switched/切换, got %q", m2.Messages[0])
	}

	// Path set but file does not exist: ReadRecent returns nil; switched line then blank
	next2, _ := m2.Update(SessionSwitchedMsg{Path: filepath.Join(t.TempDir(), "nonexistent.jsonl")})
	m3 := next2.(Model)
	if m3.Context.CurrentSessionPath == "" {
		t.Error("CurrentSessionPath should be set when Path is non-empty")
	}
	if len(m3.Messages) < 1 {
		t.Errorf("expected at least 1 message when file missing, got %d", len(m3.Messages))
	}
	if !strings.Contains(m3.Messages[0], "Switched") && !strings.Contains(m3.Messages[0], "切换") {
		t.Errorf("first message should be switched hint, got %q", m3.Messages[0])
	}
}
