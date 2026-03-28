package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func TestBlackboxSlashRemoteOnOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/access New")
	if !got.Overlay.Active {
		t.Fatalf("expected /access New to open add-remote overlay")
	}
}

func TestBlackboxOverlayEscRunsFeatureResetHook(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/access New")
	if !m.Overlay.Active {
		t.Fatalf("precondition failed: add-remote overlay should be active")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	if got.Overlay.Active {
		t.Fatalf("expected esc to close overlay and reset feature state")
	}
}

func TestBlackboxEscSendsCancelRequest(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	next, _ := f.model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	if got.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to be cleared after Esc")
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel request to be sent")
	}
}

func TestBlackboxEscWhenIdleDoesNothing(t *testing.T) {
	f := newBlackboxFixture(t)
	next, _ := f.model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	select {
	case <-f.cancelRequest:
		t.Fatalf("did not expect cancel request while idle")
	default:
	}
	if got.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to remain false while idle")
	}
}

func TestBlackboxEscClearsSlashInputBeforeCancelling(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	m := f.model
	m.Input.SetValue("/access")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := next.(ui.Model)
	if strings.TrimSpace(m2.Input.Value()) != "" {
		t.Fatalf("expected Esc to clear slash input first, got %q", m2.Input.Value())
	}
	if !m2.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to remain true after clearing slash input")
	}
	select {
	case <-f.cancelRequest:
		t.Fatalf("did not expect cancel signal while clearing slash input")
	default:
	}

	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := next2.(ui.Model)
	if m3.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag false after second Esc")
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel signal on second Esc")
	}
}
