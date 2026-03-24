package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	m = m.CloseOverlayVisual()
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
	m = m.OpenOverlay(msg.Title, msg.Content)
	m.Overlay.Viewport = viewport.New(m.Layout.Width-4, min(m.Layout.Height-6, 20))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
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
		m.Overlay.Viewport, cmd = m.Overlay.Viewport.Update(msg)
		return m, cmd, true
	}
}
