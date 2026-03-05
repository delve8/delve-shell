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

func TestApprovalCard_ApproveYClearsPending(t *testing.T) {
	ch := make(chan bool, 1)
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	// simulate user pressing y
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m2 := next.(Model)
	if m2.Pending != nil {
		t.Error("pending should be cleared after y")
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

func TestApprovalCard_ApproveNClearsPendingAndSendsFalse(t *testing.T) {
	ch := make(chan bool, 1)
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m2 := next.(Model)
	if m2.Pending != nil {
		t.Error("pending should be cleared after n")
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
