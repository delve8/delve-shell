package ui

import "delve-shell/internal/i18n"

func (m Model) openHelpOverlay() Model {
	m = m.OpenOverlay(i18n.T(m.getLang(), i18n.KeyHelpTitle), i18n.T(m.getLang(), i18n.KeyHelpText))
	m = m.InitOverlayViewport()
	return m
}
