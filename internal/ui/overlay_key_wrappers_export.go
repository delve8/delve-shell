package ui

import tea "github.com/charmbracelet/bubbletea"

// HandleOverlayKeyDelegated runs the internal overlay key handler while
// temporarily disabling overlayKeyProviders to avoid recursive provider calls.
// This is mainly used by feature packages that want to delegate remote overlay
// key handling back to internal/ui without re-implementing all overlay logic.
func (m Model) HandleOverlayKeyDelegated(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	prev := overlayKeyProviders
	overlayKeyProviders = nil
	defer func() { overlayKeyProviders = prev }()
	return m.handleOverlayKey(key, msg)
}
