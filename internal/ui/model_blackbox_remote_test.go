package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config/llm"
	"delve-shell/internal/ui"
)

func TestBlackboxSlashUpdateSkillEnterDoesNotSilentlyDrop(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/config update-skill x")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(ui.Model)
	if got.Input.Value() == "" && !got.Overlay.Active {
		t.Fatalf("expected either overlay opened or non-empty input after enter")
	}
}

func TestBlackboxSlashNewSubmitsCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/new")
	select {
	case <-f.sessionNew:
	default:
		t.Fatalf("expected /new to emit session-new intent")
	}
	if got.Input.Value() != "" {
		t.Fatalf("expected input to be cleared after /new, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashHistoryPrefixPreviewThenEnterSwitches(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/history demo")
	select {
	case id := <-f.historyPreview:
		if id != "demo" {
			t.Fatalf("expected HistoryPreviewOpen id demo, got %q", id)
		}
	default:
		t.Fatalf("expected /history <id> to emit history preview intent")
	}
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after prefix slash execution, got %q", got.Input.Value())
	}
	withOverlay, _ := got.Update(ui.HistoryPreviewOverlayMsg{SessionID: "demo", Title: "H", Content: "preview\n\nfooter"})
	m2 := withOverlay.(ui.Model)
	if !m2.Overlay.Active {
		t.Fatalf("expected overlay after HistoryPreviewOverlayMsg")
	}
	next, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next.(ui.Model)
	select {
	case sessionID := <-f.sessionSwitch:
		if sessionID != "demo" {
			t.Fatalf("expected session switch demo, got %q", sessionID)
		}
	default:
		t.Fatalf("expected Enter in preview to emit session-switch intent")
	}
	if m3.Overlay.Active {
		t.Fatalf("expected overlay closed after confirm")
	}
}

func TestBlackboxStartupOverlayProviderOpensConfigLLM(t *testing.T) {
	open := true
	m := ui.NewModel(nil, testReadModel{openConfigLLM: &open})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(ui.Model)
	if !got.Overlay.Active || !configllm.OverlayActive() {
		t.Fatalf("expected startup overlay provider to open config model overlay")
	}
}
