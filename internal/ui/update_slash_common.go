package ui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func init() {
	registerSlashExact("/help", SlashExactDispatchEntry{
		Handle: func(m Model) (Model, tea.Cmd) {
			return m.openHelpOverlay(), nil
		},
		ClearInput: true,
	})
}

func (m Model) openHelpOverlay() Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyHelpTitle)
	m.OverlayContent = i18n.T(m.getLang(), i18n.KeyHelpText)
	m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
	m.OverlayViewport.SetContent(m.OverlayContent)
	return m
}
