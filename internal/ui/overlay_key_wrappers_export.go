package ui

import tea "github.com/charmbracelet/bubbletea"

// HandleAddSkillOverlayKey is a thin exported wrapper around the internal
// add-skill overlay key handler, so feature packages can delegate key input.
func (m Model) HandleAddSkillOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	return m.handleAddSkillOverlayKey(key, msg)
}

// HandleUpdateSkillOverlayKey is a thin exported wrapper around the internal
// update-skill overlay key handler, so feature packages can delegate key input.
func (m Model) HandleUpdateSkillOverlayKey(key string) (Model, tea.Cmd, bool) {
	return m.handleUpdateSkillOverlayKey(key)
}

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
