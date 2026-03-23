package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"delve-shell/internal/i18n"
)

func (m Model) openHelpOverlay() Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyHelpTitle)
	m.OverlayContent = i18n.T(m.getLang(), i18n.KeyHelpText)
	m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
	m.OverlayViewport.SetContent(m.OverlayContent)
	return m
}

func (m Model) openAddRemoteOverlay(save, connect bool) Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T(m.getLang(), i18n.KeyAddRemoteTitle)
	m.AddRemoteActive = true
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.AddRemoteSave = save
	m.AddRemoteConnect = connect
	m.PathCompletionCandidates = nil
	m.PathCompletionIndex = -1
	m.AddRemoteFieldIndex = 0
	m.AddRemoteHostInput = textinput.New()
	m.AddRemoteHostInput.Placeholder = "host or host:22"
	m.AddRemoteHostInput.Focus()
	m.AddRemoteUserInput = textinput.New()
	m.AddRemoteUserInput.Placeholder = "e.g. root"
	m.AddRemoteUserInput.SetValue("root")
	m.AddRemoteNameInput = textinput.New()
	m.AddRemoteNameInput.Placeholder = "name (optional)"
	m.AddRemoteKeyInput = textinput.New()
	m.AddRemoteKeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
	return m
}
