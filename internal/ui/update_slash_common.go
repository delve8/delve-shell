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
	m.Overlay.Active = true
	m.Overlay.Title = i18n.T(m.getLang(), i18n.KeyHelpTitle)
	m.Overlay.Content = i18n.T(m.getLang(), i18n.KeyHelpText)
	m.Overlay.Viewport = viewport.New(m.Width-4, min(m.Height-6, 20))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
	return m
}
