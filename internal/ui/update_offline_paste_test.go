package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

func TestHandleOfflinePasteShowMsgAppendsClipboardFeedbackToTranscript(t *testing.T) {
	old := offlinePasteClipboardWrite
	offlinePasteClipboardWrite = func(string) error { return nil }
	defer func() { offlinePasteClipboardWrite = old }()

	i18n.SetLang("en")

	m := NewModel(nil, nil)
	m.layout.Width = 80

	next, _ := m.Update(OfflinePasteShowMsg{
		Pending: &uivm.PendingOfflinePaste{
			Command: "kubectl get pods -A",
			Respond: func(string, bool) {},
		},
	})
	m2 := next.(*Model)

	joined := strings.Join(m2.messages, "\n")
	if !strings.Contains(joined, "kubectl get pods -A") {
		t.Fatalf("missing offline command in transcript: %q", joined)
	}
	if !strings.Contains(joined, i18n.T(i18n.KeyOfflinePasteCopied)) {
		t.Fatalf("missing clipboard feedback in transcript: %q", joined)
	}
}

func TestHandleOfflinePasteKeyMsgEmptyEnterKeepsPasteOpen(t *testing.T) {
	old := offlinePasteClipboardWrite
	offlinePasteClipboardWrite = func(string) error { return nil }
	defer func() { offlinePasteClipboardWrite = old }()

	i18n.SetLang("en")

	called := false
	m := NewModel(nil, nil)
	m.layout.Width = 80
	next, _ := m.Update(OfflinePasteShowMsg{
		Pending: &uivm.PendingOfflinePaste{
			Command: "kubectl get pods -A",
			Respond: func(string, bool) { called = true },
		},
	})
	m = next.(*Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*Model)

	if called {
		t.Fatal("empty Enter should not submit offline paste")
	}
	if m.ChoiceCard.offlinePaste == nil {
		t.Fatal("offline paste should remain open after empty Enter")
	}
	if got := m.ChoiceCard.offlinePaste.submitFeedback; got != i18n.T(i18n.KeyOfflinePasteEmpty) {
		t.Fatalf("submitFeedback=%q want %q", got, i18n.T(i18n.KeyOfflinePasteEmpty))
	}
}
