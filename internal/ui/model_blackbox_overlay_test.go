package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
)

func TestBlackboxSlashRemoteOnOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/access New")
	if !got.Overlay.Active {
		t.Fatalf("expected /access New to open add-remote overlay")
	}
	if got.Overlay.Title != "New Remote" {
		t.Fatalf("expected add-remote overlay title, got %q", got.Overlay.Title)
	}
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/access New") {
		t.Fatalf("expected user echo for /access New, got %q", transcript)
	}
}

func TestBlackboxOverlayEscRunsFeatureResetHook(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/access New")
	if !m.Overlay.Active {
		t.Fatalf("precondition failed: add-remote overlay should be active")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(*ui.Model)
	if got.Overlay.Active {
		t.Fatalf("expected esc to close overlay and reset feature state")
	}
}

func TestBlackboxOverlayEscKeepsSingleTranscriptEcho(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/access New")
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(*ui.Model)
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if strings.Count(transcript, "/access New") != 1 {
		t.Fatalf("expected one preserved user echo after overlay close, got %q", transcript)
	}
}

func TestBlackboxRemoteConnectingOverlaySwallowsKeysExceptEsc(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/access root@example.com")
	if m.Overlay.Title != "Connect Remote" {
		t.Fatalf("expected connect overlay title, got %q", m.Overlay.Title)
	}
	if !strings.Contains(m.View(), "Target: root@example.com") {
		t.Fatalf("expected connect overlay target, got %q", m.View())
	}
	select {
	case <-f.remoteOn:
	default:
		t.Fatal("expected initial AccessRemote intent")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*ui.Model)
	if !got.Overlay.Active {
		t.Fatal("expected overlay to remain active while connecting")
	}
	select {
	case target := <-f.remoteOn:
		t.Fatalf("did not expect a second AccessRemote intent, got %q", target)
	default:
	}
}

func TestBlackboxRemoteConnectFailureStaysInOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/access root@example.com")
	select {
	case <-f.remoteOn:
	default:
		t.Fatal("expected initial AccessRemote intent")
	}
	next, _ := m.Update(remote.ConnectDoneMsg{Success: false, Err: "connection refused"})
	got := next.(*ui.Model)
	if !got.Overlay.Active {
		t.Fatal("expected overlay to remain active on connect failure")
	}
	if got.Overlay.Title != "Connect Remote" {
		t.Fatalf("expected connect overlay title after failure, got %q", got.Overlay.Title)
	}
	if !strings.Contains(got.View(), "connection refused") {
		t.Fatalf("expected connect error to stay in overlay, got %q", got.View())
	}
	if strings.Contains(got.View(), "Host (address or host:port):") {
		t.Fatalf("did not expect add-remote form after connect failure, got %q", got.View())
	}
}

func TestBlackboxEscSendsCancelRequest(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	next, _ := f.model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(*ui.Model)
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
	got := next.(*ui.Model)
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
	m2 := next.(*ui.Model)
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
	m3 := next2.(*ui.Model)
	if m3.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag false after second Esc")
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel signal on second Esc")
	}
}
