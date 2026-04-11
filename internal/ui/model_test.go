package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"delve-shell/internal/ui/uivm"
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
	m.WithTranscriptLines([]string{"hello"})
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > m.layout.Height {
		t.Fatalf("View() must not exceed Height: got %d lines, Height=%d", len(lines), m.layout.Height)
	}
	if (strings.Contains(lines[0], "[IDLE]") || strings.Contains(lines[0], "[空闲]")) &&
		(strings.Contains(lines[0], "Local") || strings.Contains(lines[0], "本地")) {
		t.Error("View() should not render the footer status line at the top")
	}
	tailStart := len(lines) - 5
	if tailStart < 0 {
		tailStart = 0
	}
	footer := strings.Join(lines[tailStart:], "\n")
	if !strings.Contains(footer, "[IDLE]") && !strings.Contains(footer, "[空闲]") && !strings.Contains(footer, "[PROCESSING]") && !strings.Contains(footer, "[处理中]") {
		t.Error("View() should show status in the footer (e.g. [IDLE] or [空闲])")
	}
	if !strings.Contains(footer, "Local") && !strings.Contains(footer, "本地") {
		t.Error("View() should show remote segment in the footer (e.g. Local)")
	}

	// Small height path: footer must still appear.
	m.layout.Height = 4
	viewSmall := m.View()
	if !strings.Contains(viewSmall, "Local") && !strings.Contains(viewSmall, "本地") {
		t.Error("View() at small height should still show the footer with remote segment")
	}

	// With Pending, footer shows [NEED APPROVAL] or [待确认] (card body is in messages; align printed count for padding).
	mPending := NewModel(nil, nil)
	mPending.layout.Width = 80
	mPending.layout.Height = 24
	n, _ := mPending.Update(ChoiceCardShowMsg{PendingApproval: &uivm.PendingApproval{Command: "ls", Respond: func(uivm.ApprovalResponse) {}}})
	mm := n.(*Model)
	mm.printedMessages = len(mm.messages)
	viewPending := mm.View()
	if !strings.Contains(viewPending, "[NEED APPROVAL]") && !strings.Contains(viewPending, "[待确认]") {
		t.Error("View() with Pending should show pending status in the footer")
	}

	// Critical: with choice mode (max 3 options) and a small Height, total lines must not exceed Height,
	// so the footer stays on screen when terminal displays one full screen.
	m2 := NewModel(nil, nil)
	m2.layout.Height = 12
	m2.layout.Width = 80
	n2, _ := m2.Update(ChoiceCardShowMsg{PendingSensitive: &uivm.PendingSensitive{Command: "cat /etc/shadow", Respond: func(uivm.SensitiveChoice) {}}})
	m2 = n2.(*Model)
	m2.printedMessages = len(m2.messages)
	viewChoice := m2.View()
	choiceLines := strings.Split(viewChoice, "\n")
	if len(choiceLines) > m2.layout.Height {
		t.Errorf("View() in choice mode (3 options) must not exceed Height: got %d lines, Height=%d (footer would scroll off)", len(choiceLines), m2.layout.Height)
	}
	// Footer title must be in the visible area near the bottom, not at the top.
	visible := strings.Join(choiceLines[:min(len(choiceLines), m2.layout.Height)], "\n")
	if (strings.Contains(choiceLines[0], "[NEED APPROVAL]") || strings.Contains(choiceLines[0], "[待确认]")) &&
		(strings.Contains(choiceLines[0], "Local") || strings.Contains(choiceLines[0], "本地")) {
		t.Error("footer line should not appear as the first visible line")
	}
	if !strings.Contains(visible, "Local") && !strings.Contains(visible, "本地") {
		t.Error("footer (remote segment) must appear in visible area")
	}
	if !strings.Contains(visible, "[NEED APPROVAL]") && !strings.Contains(visible, "[待确认]") {
		t.Error("footer (pending status) must appear in visible area")
	}
}

func TestNewModelInitialTranscriptOnly(t *testing.T) {
	m := NewModel([]string{"restored"}, nil)
	if len(m.messages) != 1 || m.messages[0] != "restored" {
		t.Fatalf("expected only restored line, got %#v", m.messages)
	}
}

func TestNewModelWithInputHistoryRecall(t *testing.T) {
	m := NewModelWithInputHistory([]string{"restored"}, []string{"first", "second"}, nil)
	m.Input.SetValue("")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(*Model)
	if got := m.Input.Value(); got != "second" {
		t.Fatalf("Up after restore want latest history line, got %q", got)
	}
}

func TestNewModelStartupTitleWhenEmpty(t *testing.T) {
	m := NewModel(nil, nil)
	if len(m.messages) != 1 {
		t.Fatalf("expected one startup line, got %d", len(m.messages))
	}
	if !strings.Contains(m.messages[0], "Delve Shell") {
		t.Fatalf("expected startup title in line: %q", m.messages[0])
	}
	if !strings.Contains(m.messages[0], uiVersionText()) {
		t.Fatalf("expected startup title to include version %q in line: %q", uiVersionText(), m.messages[0])
	}
}

// Regression: two WindowSize (or similar) Updates before transcriptPrintedMsg must not enqueue a second print batch.
func TestPrintTranscriptCmdSkipsSecondEnqueueBeforeTranscriptPrintedMsg(t *testing.T) {
	m := NewModel(nil, nil)
	cmd1 := m.printTranscriptCmd(false)
	if want := len(m.messages); m.printedMessages != want {
		t.Fatalf("printedMessages=%d want %d after scheduling print", m.printedMessages, want)
	}
	cmd2 := m.printTranscriptCmd(false)
	if cmd2 != nil {
		t.Fatalf("expected nil second print cmd, got non-nil")
	}
	_ = cmd1
}

func TestBottomChromeReserveRowsAtLeastInputChromeHeight(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	m.layout.Height = 24
	bottom := renderShortSeparator(m.layout.Width) + "\n" + m.Input.View() + m.inputBelowBlock(false) + m.footerLine()
	if got := m.bottomChromeReserveRows(bottom); got < m.inputChromeHeight() {
		t.Fatalf("bottomChromeReserveRows=%d want >= inputChromeHeight=%d", got, m.inputChromeHeight())
	}
}

func TestInputChromeHeightStableIdleVsWaitingSingleLine(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	m.Input.SetValue("")
	m.syncInputHeight()
	idleChrome := m.inputChromeHeight()
	m.Interaction.WaitingForAI = true
	m.syncInputHeight()
	waitingChrome := m.inputChromeHeight()
	if idleChrome != waitingChrome {
		t.Fatalf("inputChromeHeight idle=%d waiting=%d want equal (separator band stable)", idleChrome, waitingChrome)
	}
}

func TestMainTopPaddingLinesShrinksAsTranscriptPrints(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	m.layout.Height = 24
	bottom := renderShortSeparator(m.layout.Width) + "\n" + m.Input.View() + m.inputBelowBlock(false) + m.footerLine()

	initialPad := m.normalModeTopPaddingLines(bottom)
	if initialPad <= 0 {
		t.Fatalf("expected positive initial top padding, got %d", initialPad)
	}

	m.AppendTranscriptLines("line1", "line2", "line3")
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
	// Replace default startup banner so printed line count stays small (padding math test).
	line := strings.Repeat("x", 60)
	wide.WithTranscriptLines([]string{line})
	wide.printedMessages = len(wide.messages)
	wideBottom := renderShortSeparator(wide.layout.Width) + "\n" + wide.Input.View() + wide.inputBelowBlock(false) + wide.footerLine()

	narrow := NewModel(nil, nil)
	narrow.layout.Width = 20
	narrow.layout.Height = 24
	narrow.WithTranscriptLines([]string{line})
	narrow.printedMessages = len(narrow.messages)
	narrowBottom := renderShortSeparator(narrow.layout.Width) + "\n" + narrow.Input.View() + narrow.inputBelowBlock(false) + narrow.footerLine()

	if narrow.normalModeTopPaddingLines(narrowBottom) >= wide.normalModeTopPaddingLines(wideBottom) {
		t.Fatalf("expected narrower terminal to leave less top padding, wide=%d narrow=%d", wide.normalModeTopPaddingLines(wideBottom), narrow.normalModeTopPaddingLines(narrowBottom))
	}
}

// Regression: second approval card in one session must reset viewport + choice state (first card OK, second broken).
func TestTwoSequentialApprovalCards(t *testing.T) {
	m := NewModel(nil, nil)
	var mm *Model
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	mm = next.(*Model)

	var approved1, approved2 int
	p1 := &uivm.PendingApproval{
		Command: "first-cmd",
		Respond: func(r uivm.ApprovalResponse) {
			if r.Approved {
				approved1++
			}
		},
	}
	next, _ = mm.Update(ChoiceCardShowMsg{PendingApproval: p1})
	mm = next.(*Model)
	next, cmd := mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if cmd == nil {
		t.Fatal("expected tea.Cmd to flush decision lines to scrollback immediately after approve")
	}
	mm = next.(*Model)
	if approved1 != 1 {
		t.Fatalf("first card: want approve callback once, got %d", approved1)
	}
	if mm.ChoiceCard.pending != nil {
		t.Fatal("first card: pending should be cleared")
	}
	if mm.Interaction.ChoiceIndex != 0 {
		t.Fatalf("after first card: ChoiceIndex=%d want 0", mm.Interaction.ChoiceIndex)
	}

	p2 := &uivm.PendingApproval{
		Command: "second-cmd",
		Respond: func(r uivm.ApprovalResponse) {
			if r.Approved {
				approved2++
			}
		},
	}
	next, _ = mm.Update(ChoiceCardShowMsg{PendingApproval: p2})
	mm = next.(*Model)
	if mm.Interaction.ChoiceIndex != 0 {
		t.Fatalf("second card show: ChoiceIndex=%d want 0", mm.Interaction.ChoiceIndex)
	}
	joined := strings.Join(mm.messages, "\n")
	if !strings.Contains(joined, "second-cmd") {
		t.Fatalf("transcript should include second card command; got snippet: %q", truncateForTest(joined, 400))
	}
	next, _ = mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm = next.(*Model)
	if approved2 != 1 {
		t.Fatalf("second card: want approve via Enter once, got %d", approved2)
	}
}

func truncateForTest(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

type stubReadModelRemote struct {
	active  bool
	label   string
	offline bool
}

func (stubReadModelRemote) TakeOpenConfigModelOnFirstLayout() bool { return false }

func (s stubReadModelRemote) OfflineExecutionMode() bool { return s.offline }

func (s stubReadModelRemote) InitialRemoteFooter() (active bool, label string, offline bool) {
	return s.active, s.label, s.offline
}

func TestNewModelSeedsRemoteFromReadModel(t *testing.T) {
	m := NewModel(nil, stubReadModelRemote{active: true, label: "prod (10.0.0.1)", offline: false})
	if !m.Remote.Active || m.Remote.Label != "prod (10.0.0.1)" || m.Remote.Offline {
		t.Fatalf("Remote: %+v", m.Remote)
	}
	m2 := NewModel(nil, stubReadModelRemote{active: false, offline: true})
	if m2.Remote.Active || !m2.Remote.Offline {
		t.Fatalf("offline Remote: %+v", m2.Remote)
	}
}

func TestBashReturnTranscriptLineNonEmpty(t *testing.T) {
	s := BashReturnTranscriptLine()
	if s == "" {
		t.Fatal("BashReturnTranscriptLine should be non-empty")
	}
	if !strings.Contains(s, "Info:") {
		t.Fatalf("expected Info prefix in styled line: %q", s)
	}
}

func TestAppendInfoNotice_AppendsStyledLineAndBlank(t *testing.T) {
	m := NewModel(nil, nil)
	before := len(m.messages)
	m.AppendInfoNotice("saved")
	if len(m.messages) != before+2 {
		t.Fatalf("messages=%d want %d", len(m.messages), before+2)
	}
	if !strings.Contains(m.messages[before], "Info:") || !strings.Contains(m.messages[before], "saved") {
		t.Fatalf("unexpected info line: %q", m.messages[before])
	}
	if m.messages[before+1] != "" {
		t.Fatalf("want trailing blank line, got %q", m.messages[before+1])
	}
}

func TestRenderTranscriptLines_LineResultFitsWidthWithANSI(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 48
	long := "\x1b[34m" + strings.Repeat("W", 120) + "\x1b[0m"
	out := m.renderTranscriptLines([]uivm.Line{{Kind: uivm.LineResult, Text: long}})
	if len(out) < 2 {
		t.Fatalf("expected wrapped result into multiple rows, got %d lines", len(out))
	}
	for i, ln := range out {
		if sw := ansi.StringWidth(ln); m.contentWidth() > 0 && sw > m.contentWidth() {
			t.Fatalf("line %d: display width %d > content width %d", i, sw, m.contentWidth())
		}
	}
}

func TestFormatUserTranscriptLinesDoesNotPadStoredRowsToFullWidth(t *testing.T) {
	lines := formatUserTranscriptLines("> ", "hello", 20)
	if len(lines) != 1 {
		t.Fatalf("line count=%d want 1", len(lines))
	}
	if got := ansi.StringWidth(lines[0]); got != shortSeparatorDisplayWidth(20) {
		t.Fatalf("stored user row width=%d want %d", got, shortSeparatorDisplayWidth(20))
	}
	if got := ansi.StringWidth(lines[0]); got >= 20 {
		t.Fatalf("stored user row width=%d must stay below full width", got)
	}
}

func TestFormatUserTranscriptLinesDoesNotPadWhenBlockExceedsSeparatorWidth(t *testing.T) {
	lines := formatUserTranscriptLines("> ", strings.Repeat("x", 12), 20)
	if len(lines) != 1 {
		t.Fatalf("line count=%d want 1", len(lines))
	}
	if got := ansi.StringWidth(lines[0]); got != len("> ")+12 {
		t.Fatalf("stored user row width=%d want %d", got, len("> ")+12)
	}
}

func TestShortSeparatorDisplayWidthCapsAt80(t *testing.T) {
	if got := shortSeparatorDisplayWidth(400); got != 80 {
		t.Fatalf("shortSeparatorDisplayWidth(400)=%d want 80", got)
	}
}

func TestAppendTranscriptUserLinesTreatsOldSeparatorWidthAsSameSeparator(t *testing.T) {
	oldSep := renderShortSeparator(20)
	got := appendTranscriptUserLines([]string{"existing", oldSep}, "> ", "hello", 120)
	count := 0
	for _, line := range got {
		if isRenderedShortSeparator(line) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("separator count=%d want 1 in %#v", count, got)
	}
}

func TestAppendUserTranscriptLineKeepsSingleSeparator(t *testing.T) {
	m := NewModel(nil, nil)
	m.WithTranscriptLines([]string{"existing", renderShortSeparator(20)})
	m.appendUserTranscriptLine("hello")
	count := 0
	for _, line := range m.messages {
		if isRenderedShortSeparator(line) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("separator count=%d want 1 in %#v", count, m.messages)
	}
}

func TestAppendSuggestedLineRendersHintWithoutInfoPrefix(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	m.WithTranscriptLines(nil)
	m.appendSuggestedLine("echo hi")
	if len(m.messages) < 2 {
		t.Fatalf("want run line + hint, got %#v", m.messages)
	}
	if strings.Contains(ansi.Strip(m.messages[1]), "Info:") {
		t.Fatalf("suggested hint should not gain info prefix: %q", m.messages[1])
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

func TestPrintedTranscriptLineCountUsesScreenTranscriptStart(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 20
	m.WithTranscriptLines([]string{
		"1111111111",
		"2222222222",
		"3333333333",
		"4444444444",
	})
	m.printedMessages = len(m.messages)
	m.screenTranscriptStart = 2

	if got := m.printedTranscriptLineCount(); got != 2 {
		t.Fatalf("printedTranscriptLineCount=%d want 2 from replayed tail only", got)
	}
}

func TestRecentTranscriptReplayStartCapsToLatest100k(t *testing.T) {
	end := maxFullTranscriptReplayLines + 5
	if got := recentTranscriptReplayStart(0, end); got != 5 {
		t.Fatalf("recentTranscriptReplayStart=%d want 5", got)
	}
	if got := recentTranscriptReplayStart(10, end); got != 10 {
		t.Fatalf("recentTranscriptReplayStart must not move before explicit start, got %d", got)
	}
}

func TestOverlayInnerWidthUsesWiderModal(t *testing.T) {
	if got := overlayInnerWidth(120); got != 108 {
		t.Fatalf("overlayInnerWidth(120)=%d want 108", got)
	}
	if got := overlayInnerWidth(200); got != 116 {
		t.Fatalf("overlayInnerWidth(200)=%d want 116", got)
	}
}

func TestOverlayViewportHeightUsesAvailableScreenHeight(t *testing.T) {
	if got := overlayViewportHeight(40, ""); got != 34 {
		t.Fatalf("overlayViewportHeight(40)=%d want 34", got)
	}
	footer := "line1\nline2"
	if got := overlayViewportHeight(40, footer); got != 31 {
		t.Fatalf("overlayViewportHeight(40, footer)=%d want 31", got)
	}
}

func TestPrintTranscriptFromCmdUsesBulkWriteChunks(t *testing.T) {
	m := NewModel(nil, nil)
	lines := make([]string, transcriptBulkPrintChunkLines*2+1)
	for i := range lines {
		lines[i] = strings.Repeat("z", 18)
	}
	m.WithTranscriptLines(lines)
	cmd := m.printTranscriptFromCmd(0, false)
	if cmd == nil {
		t.Fatal("expected non-nil print cmd")
	}
	msg := cmd()
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Slice {
		t.Fatalf("expected sequence message slice, got %T", msg)
	}
	wantCmds := 3 + 1 // three bulk prints + transcriptPrintedMsg
	if got := v.Len(); got != wantCmds {
		t.Fatalf("bulk print cmd count=%d want %d", got, wantCmds)
	}
}

func TestClearScrollbackCmdEmitsEraseEntireDisplay(t *testing.T) {
	cmd := clearScrollbackCmd()
	if cmd == nil {
		t.Fatal("expected clear scrollback cmd")
	}
	msg := cmd()
	v := reflect.ValueOf(msg)
	if v.Kind() != reflect.Struct {
		t.Fatalf("expected printLineMessage struct, got %T", msg)
	}
	body := v.FieldByName("messageBody")
	if !body.IsValid() || body.Kind() != reflect.String {
		t.Fatalf("expected messageBody field on %T", msg)
	}
	got := body.String()
	if !strings.Contains(got, ansi.EraseEntireDisplay) {
		t.Fatalf("clearScrollbackCmd body %q missing erase-entire-display sequence", got)
	}
}

func TestReplayTruncatedNoticeLinesShownOnlyWhenCapped(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 80
	if got := m.replayTruncatedNoticeLines(0); got != nil {
		t.Fatalf("expected no replay notice without truncation, got %#v", got)
	}
	got := m.replayTruncatedNoticeLines(1)
	if len(got) != 3 {
		t.Fatalf("expected three notice lines, got %d", len(got))
	}
	if !strings.Contains(ansi.Strip(got[1]), "/history") {
		t.Fatalf("expected replay notice to mention /history, got %q", got[1])
	}
}

func TestFinalizeUpdateOverlayCloseReplaysLatest100kLines(t *testing.T) {
	m := NewModel(nil, nil)
	lines := make([]string, maxFullTranscriptReplayLines+5)
	for i := range lines {
		lines[i] = "line"
	}
	m.WithTranscriptLines(lines)
	m.printedMessages = len(m.messages)
	m.Overlay.Active = false

	_, cmd := finalizeUpdate(true, m, nil)
	if cmd == nil {
		t.Fatal("expected overlay close to schedule replay")
	}
	if m.screenTranscriptStart != 5 {
		t.Fatalf("screenTranscriptStart=%d want 5", m.screenTranscriptStart)
	}
	if m.screenPrefixRows <= 0 {
		t.Fatalf("screenPrefixRows=%d want > 0 for truncated replay banner", m.screenPrefixRows)
	}
	if strings.Contains(strings.Join(m.messages, "\n"), "/history") {
		t.Fatalf("replay truncated notice must not be stored in transcript: %#v", m.messages)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
