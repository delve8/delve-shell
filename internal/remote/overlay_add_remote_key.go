package remote

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

// addRemoteFieldCount returns the number of focusable fields in the Add remote form.
// Indices: 0 Host, 1 User, 2 Key path, 3 Save row; when Save: 4 Name.
func addRemoteFieldCount(s AddRemoteOverlayState) int {
	if s.Save {
		return 5
	}
	return 4
}

func applyAddRemoteFieldFocus(state *AddRemoteOverlayState) {
	state.HostInput.Blur()
	state.UserInput.Blur()
	state.NameInput.Blur()
	state.KeyInput.Blur()
	switch state.FieldIndex {
	case 0:
		state.HostInput.Focus()
	case 1:
		state.UserInput.Focus()
	case 2:
		state.KeyInput.Focus()
	case 3:
		// Save row — no text field
	case 4:
		state.NameInput.Focus()
	}
}

func moveAddRemoteField(state *AddRemoteOverlayState, dir int, pcState *pathcomplete.State) {
	fieldCount := addRemoteFieldCount(*state)
	state.FieldIndex = (state.FieldIndex + dir + fieldCount) % fieldCount
	applyAddRemoteFieldFocus(state)
	if state.FieldIndex != 2 {
		pcState.Candidates = nil
		pcState.Index = -1
	} else {
		pcState.Candidates = pathcomplete.Candidates(state.KeyInput.Value())
		if len(pcState.Candidates) > 0 {
			pcState.Index = 0
		} else {
			pcState.Index = -1
		}
	}
}

func firstIncompleteAddRemoteField(state AddRemoteOverlayState) (idx int, missing bool) {
	if strings.TrimSpace(state.HostInput.Value()) == "" {
		return 0, true
	}
	if strings.TrimSpace(state.UserInput.Value()) == "" {
		return 1, true
	}
	return 0, false
}

func handleAddRemoteOverlayKey(m *ui.Model, key string, msg tea.KeyMsg) (*ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	ret := func(model *ui.Model, cmd tea.Cmd, handled bool) (*ui.Model, tea.Cmd, bool) {
		setRemoteOverlayState(state)
		return model, cmd, handled
	}

	if state.AddRemote.OfferOverwrite {
		switch key {
		case teakey.Up, teakey.Down:
			state.AddRemote.ChoiceIndex = remoteAuthToggleChoice(state.AddRemote.ChoiceIndex)
			return ret(m, nil, true)
		case teakey.Enter:
			if state.AddRemote.ChoiceIndex == 0 {
				return handleAddRemoteOverwriteConfirm(m, state, ret)
			}
			state.AddRemote.Error = ""
			state.AddRemote.OfferOverwrite = false
			state.AddRemote.ChoiceIndex = 0
			return ret(m, nil, true)
		case "1":
			state.AddRemote.ChoiceIndex = 0
			return handleAddRemoteOverwriteConfirm(m, state, ret)
		case "2":
			state.AddRemote.Error = ""
			state.AddRemote.OfferOverwrite = false
			state.AddRemote.ChoiceIndex = 0
			return ret(m, nil, true)
		}
	}

	switch key {
	case teakey.Tab:
		if state.AddRemote.FieldIndex == 2 {
			cands := pcState.Candidates
			if len(cands) > 0 && pcState.Index >= 0 && pcState.Index < len(cands) {
				chosen := cands[pcState.Index]
				state.AddRemote.KeyInput.SetValue(chosen)
				state.AddRemote.KeyInput.CursorEnd()
				if strings.HasSuffix(chosen, "/") {
					pcState.Candidates = pathcomplete.Candidates(chosen)
					pcState.Index = 0
				} else {
					pcState.Candidates = nil
					pcState.Index = -1
				}
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
		}
		moveAddRemoteField(&state.AddRemote, 1, &pcState)
		pathcomplete.SetState(pcState)
		return ret(m, nil, true)

	case teakey.Up, teakey.Down:
		if state.AddRemote.FieldIndex == 2 && len(pcState.Candidates) > 0 {
			cands := pcState.Candidates
			if key == teakey.Up {
				pcState.Index--
				if pcState.Index < 0 {
					pcState.Index = len(cands) - 1
				}
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
			if key == teakey.Down {
				pcState.Index = (pcState.Index + 1) % len(cands)
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
		}

		dir := 1
		if key == teakey.Up {
			dir = -1
		}
		moveAddRemoteField(&state.AddRemote, dir, &pcState)
		pathcomplete.SetState(pcState)
		return ret(m, nil, true)

	case " ":
		if state.AddRemote.FieldIndex == 3 {
			state.AddRemote.Save = !state.AddRemote.Save
			if state.AddRemote.Save {
				state.AddRemote.FieldIndex = 4
			}
			applyAddRemoteFieldFocus(&state.AddRemote)
			return ret(m, nil, true)
		}

	case teakey.Enter:
		if state.AddRemote.FieldIndex == 2 {
			cands := pcState.Candidates
			if len(cands) > 0 && pcState.Index >= 0 && pcState.Index < len(cands) {
				chosen := cands[pcState.Index]
				state.AddRemote.KeyInput.SetValue(chosen)
				state.AddRemote.KeyInput.CursorEnd()
				if strings.HasSuffix(chosen, "/") {
					pcState.Candidates = pathcomplete.Candidates(chosen)
					pcState.Index = 0
				} else {
					pcState.Candidates = nil
					pcState.Index = -1
				}
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
		}
		if state.AddRemote.FieldIndex != addRemoteFieldCount(state.AddRemote)-1 {
			moveAddRemoteField(&state.AddRemote, 1, &pcState)
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}

		host := strings.TrimSpace(state.AddRemote.HostInput.Value())
		user := strings.TrimSpace(state.AddRemote.UserInput.Value())
		name := strings.TrimSpace(state.AddRemote.NameInput.Value())
		keyPath := strings.TrimSpace(state.AddRemote.KeyInput.Value())

		if missingIdx, missing := firstIncompleteAddRemoteField(state.AddRemote); missing {
			state.AddRemote.FieldIndex = missingIdx
			applyAddRemoteFieldFocus(&state.AddRemote)
			if missingIdx == 0 {
				state.AddRemote.Error = "host is required"
			} else {
				state.AddRemote.Error = "username is required"
			}
			return ret(m, nil, true)
		}
		if strings.Contains(host, "@") {
			state.AddRemote.Error = "host must not contain @"
			return ret(m, nil, true)
		}

		target := user + "@" + host
		if state.AddRemote.Save {
			if err := config.AddRemote(target, name, keyPath); err != nil {
				state.AddRemote.Error = err.Error()
				state.AddRemote.OfferOverwrite = strings.Contains(err.Error(), "already exists")
				if state.AddRemote.OfferOverwrite {
					state.AddRemote.ChoiceIndex = 0
				}
				return ret(m, nil, true)
			}
			state.AddRemote.OfferOverwrite = false
			state.AddRemote.ChoiceIndex = 0
			display := host
			if name != "" {
				display = name + " (" + host + ")"
			}
			m.AppendInfoNotice(i18n.Tf(i18n.KeyConfigRemoteAdded, display))
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.ConfigUpdated{})
			}
		}
		rememberLastAddRemoteUsername(user)
		rememberLastAddRemoteIdentityFile(keyPath)
		state.AddRemote.Connecting = true
		state.AddRemote.Error = ""
		if m.CommandSender == nil || !m.CommandSender.Send(hostcmd.AccessRemote{Target: target}) {
			state.AddRemote.Connecting = false
		}
		return ret(m, nil, true)
	}

	var cmd tea.Cmd
	switch state.AddRemote.FieldIndex {
	case 0:
		state.AddRemote.HostInput, cmd = state.AddRemote.HostInput.Update(msg)
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
	case 1:
		state.AddRemote.UserInput, cmd = state.AddRemote.UserInput.Update(msg)
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
	case 2:
		state.AddRemote.KeyInput, cmd = state.AddRemote.KeyInput.Update(msg)
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
		pcState.Candidates = pathcomplete.Candidates(state.AddRemote.KeyInput.Value())
		if len(pcState.Candidates) > 0 {
			pcState.Index = 0
		} else {
			pcState.Index = -1
		}
	case 3:
		cmd = nil
	case 4:
		state.AddRemote.NameInput, cmd = state.AddRemote.NameInput.Update(msg)
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
	}
	pathcomplete.SetState(pcState)
	_ = msg
	return ret(m, cmd, true)
}

func handleAddRemoteOverwriteConfirm(
	m *ui.Model,
	state remoteOverlayState,
	ret func(*ui.Model, tea.Cmd, bool) (*ui.Model, tea.Cmd, bool),
) (*ui.Model, tea.Cmd, bool) {
	host := strings.TrimSpace(state.AddRemote.HostInput.Value())
	user := strings.TrimSpace(state.AddRemote.UserInput.Value())
	name := strings.TrimSpace(state.AddRemote.NameInput.Value())
	keyPath := strings.TrimSpace(state.AddRemote.KeyInput.Value())
	if host == "" {
		return ret(m, nil, true)
	}
	target := user + "@" + host
	if err := config.UpdateRemote(target, name, keyPath); err != nil {
		state.AddRemote.Error = err.Error()
		state.AddRemote.OfferOverwrite = false
		state.AddRemote.ChoiceIndex = 0
		return ret(m, nil, true)
	}
	display := host
	if name != "" {
		display = name + " (" + host + ")"
	}
	m.AppendInfoNotice(i18n.Tf(i18n.KeyConfigRemoteAdded, display))
	rememberLastAddRemoteUsername(user)
	rememberLastAddRemoteIdentityFile(keyPath)
	m.CloseOverlayVisual()
	state.AddRemote.Active = false
	state.AddRemote.Error = ""
	state.AddRemote.OfferOverwrite = false
	state.AddRemote.ChoiceIndex = 0
	m.Input.Focus()
	if m.CommandSender != nil {
		_ = m.CommandSender.Send(hostcmd.ConfigUpdated{})
	}
	return ret(m, nil, true)
}

func rememberLastAddRemoteIdentityFile(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	_ = config.SetLastIdentityFile(path)
}

func rememberLastAddRemoteUsername(username string) {
	username = strings.TrimSpace(username)
	if username == "" {
		return
	}
	_ = config.SetLastUsername(username)
}
