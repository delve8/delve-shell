package ui

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/history"
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
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Pending = &agent.ApprovalRequest{
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
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	// simulate user pressing 1 (approve)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m2 := next.(Model)
	if m2.Pending != nil {
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
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m2 := next.(Model)
	if m2.Pending != nil {
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
	m.Height = 24
	m.Width = 80
	view := m.View()
	// Header contains Auto-run label and a status in brackets
	if !strings.Contains(view, "[IDLE]") && !strings.Contains(view, "[空闲]") && !strings.Contains(view, "[PROCESSING]") && !strings.Contains(view, "[处理中]") {
		t.Error("View() should show status in header (e.g. [IDLE] or [空闲])")
	}
	if !strings.Contains(view, "Auto-Run") && !strings.Contains(view, "自动执行") {
		t.Error("View() should show Auto-Run label in header")
	}

	// Small height path: header must still appear first
	m.Height = 4
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "Auto-Run") && !strings.Contains(viewSmall, "自动执行") {
		t.Error("View() at small height should still show header with Auto-Run label")
	}

	// With Pending, header shows [NEED APPROVAL] or [待确认]
	ch := make(chan agent.ApprovalResponse, 1)
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.Height = 24
	viewPending := m.View()
	if !strings.Contains(viewPending, "[NEED APPROVAL]") && !strings.Contains(viewPending, "[待确认]") {
		t.Error("View() with Pending should show pending status in header")
	}

	// Critical: with choice mode (max 3 options) and a small Height, total lines must not exceed Height,
	// so the header (first 2 lines) stays on screen when terminal displays one full screen.
	m2 := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func() bool { return true }, nil, "", false)
	m2.Height = 12
	m2.Width = 80
	m2.PendingSensitive = &agent.SensitiveConfirmationRequest{Command: "cat /etc/shadow", ResponseCh: make(chan agent.SensitiveChoice, 1)}
	viewChoice := m2.View()
	lines := strings.Split(viewChoice, "\n")
	if len(lines) > m2.Height {
		t.Errorf("View() in choice mode (3 options) must not exceed Height: got %d lines, Height=%d (header would scroll off)", len(lines), m2.Height)
	}
	// First line must be the header title (Auto-Run + status)
	visible := strings.Join(lines[:min(len(lines), m2.Height)], "\n")
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
	m.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch}
	m.ChoiceIndex = 0

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if m2.Pending != nil {
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
	m3.Pending = &agent.ApprovalRequest{Command: "ls", ResponseCh: ch2}
	m3.ChoiceIndex = 1

	next2, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m4 := next2.(Model)
	if m4.Pending != nil {
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

func TestSlashDropdown_UpDownAndEnterFill(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Input.SetValue("/")
	m.Input.CursorEnd()
	if got := m.Input.Value(); got != "/" {
		t.Fatalf("precondition: expected input value '/', got %q", got)
	}

	if got := (tea.KeyMsg{Type: tea.KeyDown}).String(); got != "down" {
		t.Fatalf("unexpected KeyDown String(): %q", got)
	}

	// Down should move selection from index 0 to 1 (some other slash option).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := next.(Model)
	if got := m2.Input.Value(); got != "/" {
		t.Fatalf("expected input to remain '/', got %q", got)
	}
	if m2.SlashSuggestIndex == 0 {
		t.Fatalf("expected SlashSuggestIndex to change after Down, got %d", m2.SlashSuggestIndex)
	}

	// Enter should fill the chosen option into the input (not execute), so input is no longer just "/".
	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(Model)
	// Depending on which option is selected, Enter may execute immediately (e.g. /cancel, /q),
	// which clears input. The minimum contract is: it should not remain exactly "/".
	if strings.TrimSpace(m3.Input.Value()) == "/" {
		t.Fatalf("expected Enter to fill a slash option, got input %q", m3.Input.Value())
	}
	// If it was a fill (not execute), it must start with "/".
	if v := strings.TrimSpace(m3.Input.Value()); v != "" && !strings.HasPrefix(v, "/") {
		t.Fatalf("expected filled input to start with '/', got %q", m3.Input.Value())
	}
}

func TestSlashDropdown_UpdateSkill_EnterExecutesOverlay(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	// This command does not need to exist in manifest; openUpdateSkillOverlay shows an overlay error.
	m.Input.SetValue("/config update-skill")
	m.Input.CursorEnd()

	// Move selection so we are not always on "/config add-remote" etc; we want update-skill option itself.
	// With input "/config update-skill", dropdown should include update-skill items; Enter should execute
	// when a concrete "/config update-skill <name>" suggestion is selected.
	// Since tests have no installed skills, getSlashOptionsForInput will show a placeholder; simulate a concrete chosen option.
	// We do this by setting input to a prefix and relying on fill-only/execute logic for update-skill suggestions.
	m.Input.SetValue("/config update-skill x")
	m.Input.CursorEnd()
	// Press Enter: should not crash; may fill or execute depending on suggestions. At minimum, it should not clear input to empty silently.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if m2.Input.Value() == "" && !m2.OverlayActive {
		t.Fatalf("expected either overlay or non-empty input after Enter, got empty input and no overlay")
	}
}

func TestSlashCommand_Help_EnterOpensOverlay(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Input.SetValue("/help")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if !m2.OverlayActive {
		t.Fatalf("expected /help Enter to open overlay, OverlayActive=false")
	}
	if m2.OverlayTitle == "" {
		t.Fatalf("expected /help overlay to have a title")
	}
}

func TestOverlayEsc_CloseHooksClearFeatureFlags(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.OverlayActive = true
	m.OverlayTitle = "x"
	m.AddRemoteActive = true

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := next.(Model)
	if m2.OverlayActive {
		t.Fatal("expected overlay closed after Esc")
	}
	if m2.AddRemoteActive {
		t.Fatal("expected overlay close feature reset to clear AddRemoteActive")
	}
}

func TestSlashCommand_RemoteOn_EnterOpensOverlay(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Input.SetValue("/remote on")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if !m2.OverlayActive || !m2.AddRemoteActive {
		t.Fatalf("expected /remote on Enter to open overlay, OverlayActive=%v AddRemoteActive=%v", m2.OverlayActive, m2.AddRemoteActive)
	}
}

func TestSlashCommand_ConfigDelRemote_EnterFillsInput(t *testing.T) {
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.Input.SetValue("/config del-remote")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if strings.TrimSpace(m2.Input.Value()) != "/config del-remote" {
		t.Fatalf("expected /config del-remote Enter to keep filled command, got %q", m2.Input.Value())
	}
	if !strings.HasSuffix(m2.Input.Value(), " ") {
		t.Fatalf("expected /config del-remote to append trailing space, got %q", m2.Input.Value())
	}
}

func TestSlashDropdown_Cancel_EnterFillsThenExecutes(t *testing.T) {
	cancelCh := make(chan struct{}, 1)
	getAutoRun := func() bool { return true }
	m := NewModel(nil, nil, nil, cancelCh, nil, nil, nil, nil, nil, nil, getAutoRun, nil, "", false)
	m.WaitingForAI = true
	m.Input.SetValue("/c")
	m.Input.CursorEnd()

	// Move selection to the "/cancel" option.
	opts := getSlashOptionsForInput(m.Input.Value(), m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
	vis := visibleSlashOptions(m.Input.Value(), opts)
	for i, idx := range vis {
		if opts[idx].Cmd == "/cancel" {
			m.SlashSuggestIndex = i
			break
		}
	}

	// First Enter should fill to "/cancel", not execute.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(Model)
	if strings.TrimSpace(m2.Input.Value()) != "/cancel" {
		t.Fatalf("expected first Enter to fill input to /cancel, got %q", m2.Input.Value())
	}
	if !m2.WaitingForAI {
		t.Fatalf("expected WaitingForAI to remain true after fill-only Enter")
	}
	select {
	case <-cancelCh:
		t.Fatalf("did not expect cancel request on fill-only Enter")
	default:
	}

	// Second Enter executes /cancel.
	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(Model)
	if m3.WaitingForAI {
		t.Fatalf("expected WaitingForAI=false after executing /cancel")
	}
	select {
	case <-cancelCh:
	default:
		t.Fatalf("expected cancel request to be sent")
	}
}

// TestSessionSwitchedMsg_setsCurrentPathAndShowsSwitchedAtBottom asserts that after SessionSwitchedMsg,
// CurrentSessionPath is set and the "Switched to session" line is present (at end when there is history).
func TestSessionSwitchedMsg_setsCurrentPathAndShowsSwitchedAtBottom(t *testing.T) {
	m := NewModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, func() bool { return true }, nil, "", false)

	// Path empty: switched hint then blank line, CurrentSessionPath ""
	next, _ := m.Update(SessionSwitchedMsg{Path: ""})
	m2 := next.(Model)
	if m2.CurrentSessionPath != "" {
		t.Errorf("CurrentSessionPath should be empty when Path is empty, got %q", m2.CurrentSessionPath)
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
	if m3.CurrentSessionPath == "" {
		t.Error("CurrentSessionPath should be set when Path is non-empty")
	}
	if len(m3.Messages) < 1 {
		t.Errorf("expected at least 1 message when file missing, got %d", len(m3.Messages))
	}
	if !strings.Contains(m3.Messages[0], "Switched") && !strings.Contains(m3.Messages[0], "切换") {
		t.Errorf("first message should be switched hint, got %q", m3.Messages[0])
	}
}

// TestSessionEventsToMessages_convertsEventsToDisplayLines asserts sessionEventsToMessages produces User/AI/Run/result lines.
func TestSessionEventsToMessages_convertsEventsToDisplayLines(t *testing.T) {
	events := []history.Event{
		{Type: "user_input", Payload: json.RawMessage(`{"text":"hello"}`)},
		{Type: "llm_response", Payload: json.RawMessage(`{"reply":"hi"}`)},
		{Type: "command", Payload: json.RawMessage(`{"command":"ls","approved":true,"suggested":false}`)},
		{Type: "command_result", Payload: json.RawMessage(`{"command":"ls","stdout":"a\nb","stderr":"","exit_code":0}`)},
	}
	lines := sessionEventsToMessages(events, "en", 80)
	// Order: User, blank, AI, separator, Run, result, blank
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "User:") && !strings.Contains(lines[0], "用户") {
		t.Errorf("first line should be user label + text: %q", lines[0])
	}
	if !strings.Contains(lines[0], "hello") {
		t.Errorf("first line should contain hello: %q", lines[0])
	}
	// lines[1] is blank after user_input
	aiIdx := 2
	if !strings.Contains(lines[aiIdx], "AI:") && !strings.Contains(lines[aiIdx], "AI：") {
		t.Errorf("AI line should be AI label: %q", lines[aiIdx])
	}
	if !strings.Contains(lines[aiIdx], "hi") {
		t.Errorf("AI line should contain hi: %q", lines[aiIdx])
	}
	// command is after separator (lines[3]); result after that
	runIdx := 4
	resultIdx := 5
	if !strings.Contains(lines[runIdx], "ls") {
		t.Errorf("command line should contain ls: %q", lines[runIdx])
	}
	if !strings.Contains(lines[resultIdx], "a") || !strings.Contains(lines[resultIdx], "b") {
		t.Errorf("result line should contain stdout: %q", lines[resultIdx])
	}
}
