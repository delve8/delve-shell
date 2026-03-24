package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	m.OverlayActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	for _, h := range overlayCloseHooks {
		m = h(m)
	}
	if refocusInput {
		// Esc path keeps prior behavior: always refocus main input after closing overlays.
		m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleOverlayShowMsg(msg OverlayShowMsg) (Model, tea.Cmd) {
	m.OverlayActive = true
	m.OverlayTitle = msg.Title
	m.OverlayContent = msg.Content
	m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
	m.OverlayViewport.SetContent(m.OverlayContent)
	return m, nil
}

func (m Model) handleOverlayCloseMsg() (Model, tea.Cmd) {
	return m.closeOverlayCommon(false)
}

// handleOverlayKey routes key input when overlay is active.
func (m Model) handleOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}

	for _, p := range overlayKeyProviders {
		if m2, cmd, handled := p(m, key, msg); handled {
			return m2, cmd, true
		}
	}

	switch key {
	case "esc":
		m, cmd := m.closeOverlayCommon(true)
		return m, cmd, true
	default:
		// Generic overlay: pass up/down/pgup/pgdown.
		var cmd tea.Cmd
		m.OverlayViewport, cmd = m.OverlayViewport.Update(msg)
		return m, cmd, true
	}
}
