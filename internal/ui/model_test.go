package ui

import (
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
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	ch := make(chan bool, 1)
	m.Pending = &agent.ApprovalRequest{
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
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{
		Command:    "rm -rf /tmp/foo",
		RiskLevel:  "high",
		ResponseCh: make(chan bool, 1),
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
	ch := make(chan bool, 1)
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	// simulate user pressing 1 (approve)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m2 := next.(Model)
	if m2.Pending != nil {
		t.Error("pending should be cleared after 1")
	}
	select {
	case v := <-ch:
		if !v {
			t.Error("channel should receive true for approve")
		}
	default:
		t.Error("channel should receive true")
	}
}

func TestApprovalCard_Approve2ClearsPendingAndSendsFalse(t *testing.T) {
	ch := make(chan bool, 1)
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m2 := next.(Model)
	if m2.Pending != nil {
		t.Error("pending should be cleared after 2")
	}
	select {
	case v := <-ch:
		if v {
			t.Error("channel should receive false for reject")
		}
	default:
		t.Error("channel should receive false")
	}
}

// TestView_HeaderAlwaysShown asserts that View() always includes the header (mode + status) and that
// total output lines never exceed Height so the header stays visible when the terminal shows one screen.
func TestView_HeaderAlwaysShown(t *testing.T) {
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Height = 24
	m.Width = 80
	view := m.View()
	// Header contains mode label (en "mode" or zh "模式") and a status in brackets
	if !strings.Contains(view, "mode") && !strings.Contains(view, "模式") {
		t.Error("View() should show header with mode label (mode or 模式)")
	}
	if !strings.Contains(view, "[IDLE]") && !strings.Contains(view, "[空闲]") && !strings.Contains(view, "[PROCESSING]") && !strings.Contains(view, "[处理中]") {
		t.Error("View() should show status in header (e.g. [IDLE] or [空闲])")
	}

	// Small height path: header must still appear first
	m.Height = 4
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "mode") && !strings.Contains(viewSmall, "模式") {
		t.Error("View() at small height should still show header with mode label")
	}

	// With Pending, header shows [NEED APPROVAL] or [待确认]
	ch := make(chan bool, 1)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.Height = 24
	viewPending := m.View()
	if !strings.Contains(viewPending, "[NEED APPROVAL]") && !strings.Contains(viewPending, "[待确认]") {
		t.Error("View() with Pending should show pending status in header")
	}

	// Critical: with choice mode (max 3 options) and a small Height, total lines must not exceed Height,
	// so the header (first 2 lines) stays on screen when terminal displays one full screen.
	m2 := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m2.Height = 12
	m2.Width = 80
	m2.PendingSensitive = &agent.SensitiveConfirmationRequest{Command: "cat /etc/shadow", ResponseCh: make(chan agent.SensitiveChoice, 1)}
	viewChoice := m2.View()
	lines := strings.Split(viewChoice, "\n")
	if len(lines) > m2.Height {
		t.Errorf("View() in choice mode (3 options) must not exceed Height: got %d lines, Height=%d (header would scroll off)", len(lines), m2.Height)
	}
	// First line must be the header title (mode + status)
	visible := strings.Join(lines[:min(len(lines), m2.Height)], "\n")
	if !strings.Contains(visible, "mode") && !strings.Contains(visible, "模式") {
		t.Error("header (mode label) must appear in visible area")
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
	// Approval: ChoiceIndex 0 = approve, Enter should send true
	ch := make(chan bool, 1)
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.ChoiceIndex = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if m2.Pending != nil {
		t.Error("pending should be cleared after Enter")
	}
	select {
	case v := <-ch:
		if !v {
			t.Error("Enter with ChoiceIndex 0 should approve (true)")
		}
	default:
		t.Error("channel should receive true")
	}

	// Approval: ChoiceIndex 1 = reject, Enter should send false
	ch2 := make(chan bool, 1)
	m3 := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m3.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch2}
	m3.ChoiceIndex = 1

	next2, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := next2.(Model)
	if m4.Pending != nil {
		t.Error("pending should be cleared after Enter on option 2")
	}
	select {
	case v := <-ch2:
		if v {
			t.Error("Enter with ChoiceIndex 1 should reject (false)")
		}
	default:
		t.Error("channel should receive false")
	}
}
