package remote

import (
	"github.com/charmbracelet/bubbles/textinput"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

func openAddRemoteOverlay(m ui.Model, save, connect bool) ui.Model {
	m.OverlayActive = true
	m.OverlayTitle = i18n.T("en", i18n.KeyAddRemoteTitle)
	m.AddRemote.Active = true
	m.AddRemote.Error = ""
	m.AddRemote.OfferOverwrite = false
	m.AddRemote.Save = save
	m.AddRemote.Connect = connect
	m.PathCompletionCandidates = nil
	m.PathCompletionIndex = -1
	m.AddRemote.FieldIndex = 0
	m.AddRemote.HostInput = textinput.New()
	m.AddRemote.HostInput.Placeholder = "host or host:22"
	m.AddRemote.HostInput.Focus()
	m.AddRemote.UserInput = textinput.New()
	m.AddRemote.UserInput.Placeholder = "e.g. root"
	m.AddRemote.UserInput.SetValue("root")
	m.AddRemote.NameInput = textinput.New()
	m.AddRemote.NameInput.Placeholder = "name (optional)"
	m.AddRemote.KeyInput = textinput.New()
	m.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
	return m
}
