package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	m.OverlayActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	m.AddRemoteActive = false
	m.AddRemoteConnecting = false
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.RemoteAuthConnecting = false
	m.AddSkillActive = false
	m.AddSkillError = ""
	m.ConfigLLMActive = false
	m.ConfigLLMChecking = false
	m.ConfigLLMError = ""
	m.RemoteAuthStep = ""
	m.RemoteAuthTarget = ""
	m.RemoteAuthError = ""
	m.RemoteAuthUsername = ""
	m.UpdateSkillActive = false
	m.UpdateSkillError = ""
	if refocusInput {
		// Esc path keeps prior behavior: always refocus main input after closing overlays.
		m.Input.Focus()
	}
	return m, nil
}
