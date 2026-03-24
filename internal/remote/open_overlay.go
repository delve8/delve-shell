package remote

import (
	"github.com/charmbracelet/bubbles/textinput"

	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

func openAddRemoteOverlay(m ui.Model, save, connect bool) ui.Model {
	m = m.OpenOverlay(i18n.T("en", i18n.KeyAddRemoteTitle), "")
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.RemoteAuth = RemoteAuthOverlayState{}
	state.AddRemote.Error = ""
	state.AddRemote.OfferOverwrite = false
	state.AddRemote.Save = save
	state.AddRemote.Connect = connect
	pathcomplete.SetState(pathcomplete.State{Index: -1})
	state.AddRemote.FieldIndex = 0
	state.AddRemote.HostInput = textinput.New()
	state.AddRemote.HostInput.Placeholder = "host or host:22"
	state.AddRemote.HostInput.Focus()
	state.AddRemote.UserInput = textinput.New()
	state.AddRemote.UserInput.Placeholder = "e.g. root"
	state.AddRemote.UserInput.SetValue("root")
	state.AddRemote.NameInput = textinput.New()
	state.AddRemote.NameInput.Placeholder = "name (optional)"
	state.AddRemote.KeyInput = textinput.New()
	state.AddRemote.KeyInput.Placeholder = "~/.ssh/id_rsa (optional)"
	setRemoteOverlayState(state)
	return m
}
