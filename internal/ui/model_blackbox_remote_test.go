package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/configllm"
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

func TestBlackboxSlashSessionsPrefixSubmitsCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/sessions demo")
	select {
	case sessionID := <-f.sessionSwitch:
		if sessionID != "demo" {
			t.Fatalf("expected session switch 'demo', got %q", sessionID)
		}
	default:
		t.Fatalf("expected /sessions <id> to emit session-switch intent")
	}
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after prefix slash execution, got %q", got.Input.Value())
	}
}

func TestBlackboxStartupOverlayProviderOpensConfigLLM(t *testing.T) {
	open := true
	m := ui.NewModel(nil, testReadModel{openConfigLLM: &open})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(ui.Model)
	if !got.Overlay.Active || !configllm.OverlayActive() {
		t.Fatalf("expected startup overlay provider to open config llm overlay")
	}
}
