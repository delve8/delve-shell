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
	m = m.OpenOverlay(i18n.T(m.getLang(), i18n.KeyHelpTitle), i18n.T(m.getLang(), i18n.KeyHelpText))
	m.Overlay.Viewport = viewport.New(m.Layout.Width-4, min(m.Layout.Height-6, 20))
	m.Overlay.Viewport.SetContent(m.Overlay.Content)
	return m
}
