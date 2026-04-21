package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func TestBlackboxInputHistoryRecall(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "alpha")
	m.Interaction.WaitingForAI = false
	m = enterText(m, "beta")
	m.Interaction.WaitingForAI = false
	m.Input.SetValue("")
	m.Input.CursorEnd()

	up := func(mm *ui.Model) *ui.Model {
		next, _ := mm.Update(tea.KeyMsg{Type: tea.KeyUp})
		return next.(*ui.Model)
	}
	down := func(mm *ui.Model) *ui.Model {
		next, _ := mm.Update(tea.KeyMsg{Type: tea.KeyDown})
		return next.(*ui.Model)
	}

	m = up(m)
	if got := m.Input.Value(); got != "beta" {
		t.Fatalf("first Up want beta, got %q", got)
	}
	m = up(m)
	if got := m.Input.Value(); got != "alpha" {
		t.Fatalf("second Up want alpha, got %q", got)
	}
	m = down(m)
	if got := m.Input.Value(); got != "beta" {
		t.Fatalf("Down want beta, got %q", got)
	}
	m = down(m)
	if got := m.Input.Value(); got != "" {
		t.Fatalf("Down at end want restored empty draft, got %q", got)
	}
}

func TestBlackboxInputHistoryContinuesWhenRecalledLineIsSlash(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "plain-msg")
	m.Interaction.WaitingForAI = false
	m.Input.SetValue("/access Local")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*ui.Model)
	m.Interaction.WaitingForAI = false
	m.Input.SetValue("")
	m.Input.CursorEnd()
	up := func(mm *ui.Model) *ui.Model {
		n, _ := mm.Update(tea.KeyMsg{Type: tea.KeyUp})
		return n.(*ui.Model)
	}
	m = up(m)
	if got := m.Input.Value(); got != "/access Local" {
		t.Fatalf("first Up want latest slash line, got %q", got)
	}
	m = up(m)
	if got := m.Input.Value(); got != "plain-msg" {
		t.Fatalf("second Up while browsing must not switch to slash menu; want plain-msg, got %q", got)
	}
}

func TestBlackboxSlashSubmittedLineInInputHistory(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/access Local")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*ui.Model)
	m.Input.SetValue("")
	m.Input.CursorEnd()
	next2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next2.(*ui.Model)
	if got := m.Input.Value(); got != "/access Local" {
		t.Fatalf("Up after slash submit want line in input history, got %q", got)
	}
}

func TestBlackboxSlashInputDoesNotUseInputHistoryUp(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "chat-only")
	m.Input.SetValue("/")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = next.(*ui.Model)
	if got := m.Input.Value(); got != "/" {
		t.Fatalf("slash line should not be replaced by history, got %q", got)
	}
}

func TestBlackboxInputHistoryUpDownWhileRecalledMultiline(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "single-line")
	m.Interaction.WaitingForAI = false
	m = enterText(m, "first\nsecond")
	m.Interaction.WaitingForAI = false
	m.Input.SetValue("")
	m.Input.CursorEnd()
	up := func(mm *ui.Model) *ui.Model {
		n, _ := mm.Update(tea.KeyMsg{Type: tea.KeyUp})
		return n.(*ui.Model)
	}
	m = up(m)
	if got := m.Input.Value(); got != "first\nsecond" {
		t.Fatalf("first Up want multiline latest, got %q", got)
	}
	if m.Input.LineCount() <= 1 {
		t.Fatalf("expected multiline buffer after recall, LineCount=%d", m.Input.LineCount())
	}
	m = up(m)
	if got := m.Input.Value(); got != "single-line" {
		t.Fatalf("second Up on multiline recall must walk history, want single-line, got %q", got)
	}
}

func TestBlackboxInputHistoryContinuesWhenRecalledMultilineStartsWithSlash(t *testing.T) {
	m := ui.NewModelWithInputHistory(nil, []string{"plain-msg", "/access Local\nsecond line"}, nil)
	m.Input.SetValue("")
	m.Input.CursorEnd()
	up := func(mm *ui.Model) *ui.Model {
		n, _ := mm.Update(tea.KeyMsg{Type: tea.KeyUp})
		return n.(*ui.Model)
	}
	m = up(m)
	wantMultiline := "/access Local\nsecond line"
	if got := m.Input.Value(); got != wantMultiline {
		t.Fatalf("first Up want latest multiline slash-prefixed line, got %q", got)
	}
	if m.Input.LineCount() <= 1 {
		t.Fatalf("expected multiline buffer, LineCount=%d", m.Input.LineCount())
	}
	if !strings.HasPrefix(m.Input.Value(), "/") {
		t.Fatalf("expected buffer to start with slash for regression scenario")
	}
	m = up(m)
	if got := m.Input.Value(); got != "plain-msg" {
		t.Fatalf("second Up must walk history (not slash suggest), want plain-msg, got %q", got)
	}
}
