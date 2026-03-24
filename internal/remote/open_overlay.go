package remote

import (
	"github.com/charmbracelet/bubbles/textinput"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func openAddRemoteOverlay(m ui.Model, save, connect bool) ui.Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T("en", i18n.KeyAddRemoteTitle)
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
