package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/widget"
)

func TestTitleBarLeadingSegment(t *testing.T) {
	t.Run("default local when inactive", func(t *testing.T) {
		m := NewModel(nil, nil)
		m.Remote.Active = false
		m.Remote.Label = ""
		if got := m.titleBarLeadingSegment(); got != "Local" {
			t.Fatalf("got %q want Local", got)
		}
	})
	t.Run("remote without label", func(t *testing.T) {
		m := NewModel(nil, nil)
		m.Remote.Active = true
		m.Remote.Label = ""
		if got := m.titleBarLeadingSegment(); got != "Remote" {
			t.Fatalf("got %q want Remote", got)
		}
	})
	t.Run("remote with label", func(t *testing.T) {
		m := NewModel(nil, nil)
		m.Remote.Active = true
		m.Remote.Label = "prod"
		if got := m.titleBarLeadingSegment(); got != "Remote prod" {
			t.Fatalf("got %q want Remote prod", got)
		}
	})
	t.Run("offline overrides remote", func(t *testing.T) {
		m := NewModel(nil, nil)
		m.Remote.Active = true
		m.Remote.Label = "prod"
		m.Remote.Offline = true
		if got := m.titleBarLeadingSegment(); got != "Offline" {
			t.Fatalf("got %q want Offline", got)
		}
	})
	t.Run("offline paste uses wait input status not need approval", func(t *testing.T) {
		m := NewModel(nil, nil)
		ti := textarea.New()
		m.ChoiceCard.offlinePaste = &OfflinePasteState{Command: "true", Paste: ti}
		if got := m.statusKey(); got != i18n.KeyStatusWaitingUserInput {
			t.Fatalf("statusKey: got %q want %q", got, i18n.KeyStatusWaitingUserInput)
		}
		if got := m.titleBarStatus(); got != widget.TitleBarStatusWaitingUserInput {
			t.Fatalf("titleBarStatus: got %v want WaitingUserInput", got)
		}
	})
}
