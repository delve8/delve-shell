package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/remote/auth"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

func initRemoteAuthIdentityInput(state *remoteOverlayState) {
	state.RemoteAuth.Input = textinput.New()
	state.RemoteAuth.Input.Placeholder = i18n.T(i18n.KeyRemoteAuthIdentityPlaceholder)
	state.RemoteAuth.Input.EchoMode = textinput.EchoNormal
	if prefill := resolveRemoteIdentityPrefill(state.RemoteAuth.Target); prefill != "" {
		state.RemoteAuth.Input.SetValue(prefill)
		state.RemoteAuth.Input.CursorEnd()
	}
	state.RemoteAuth.Input.Focus()
}

func handleRemoteAuthOverlayKey(m *ui.Model, key string, msg tea.KeyMsg) (*ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	ret := func(model *ui.Model, cmd tea.Cmd, handled bool) (*ui.Model, tea.Cmd, bool) {
		setRemoteOverlayState(state)
		return model, cmd, handled
	}

	switch state.RemoteAuth.Step {
	case AuthStepHostKey:
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}
		switch key {
		case teakey.Up:
			state.RemoteAuth.ChoiceIndex = remoteAuthToggleChoice(state.RemoteAuth.ChoiceIndex)
			return ret(m, nil, true)
		case teakey.Down:
			state.RemoteAuth.ChoiceIndex = remoteAuthToggleChoice(state.RemoteAuth.ChoiceIndex)
			return ret(m, nil, true)
		case teakey.Enter:
			if state.RemoteAuth.ChoiceIndex == 0 {
				state.RemoteAuth.Connecting = true
				if m.CommandSender != nil {
					_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
						Target: state.RemoteAuth.Target,
						Kind:   remoteauth.ResponseKindHostKeyAccept,
					}})
				}
				return ret(m, nil, true)
			}
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
					Target: state.RemoteAuth.Target,
					Kind:   remoteauth.ResponseKindHostKeyReject,
				}})
			}
			cmd := m.CloseOverlayAndRefocusInput()
			state.RemoteAuth = RemoteAuthOverlayState{}
			return ret(m, cmd, true)
		case "1":
			state.RemoteAuth.Connecting = true
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
					Target: state.RemoteAuth.Target,
					Kind:   remoteauth.ResponseKindHostKeyAccept,
				}})
			}
			return ret(m, nil, true)
		case "2":
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
					Target: state.RemoteAuth.Target,
					Kind:   remoteauth.ResponseKindHostKeyReject,
				}})
			}
			cmd := m.CloseOverlayAndRefocusInput()
			state.RemoteAuth = RemoteAuthOverlayState{}
			return ret(m, cmd, true)
		}
		return ret(m, nil, true)

	case AuthStepAutoIdentity:
		return ret(m, nil, true)

	case AuthStepUsername:
		if key == teakey.Enter {
			state.RemoteAuth.Username = strings.TrimSpace(state.RemoteAuth.UsernameInput.Value())
			if state.RemoteAuth.Username == "" {
				state.RemoteAuth.Error = "username is required"
				return ret(m, nil, true)
			}
			state.RemoteAuth.Error = ""
			state.RemoteAuth.Step = AuthStepChoose
			state.RemoteAuth.ChoiceIndex = 0
			return ret(m, nil, true)
		}
		var cmd tea.Cmd
		state.RemoteAuth.UsernameInput, cmd = state.RemoteAuth.UsernameInput.Update(msg)
		state.RemoteAuth.Error = ""
		return ret(m, cmd, true)

	case AuthStepChoose:
		switch key {
		case teakey.Up:
			state.RemoteAuth.ChoiceIndex = remoteAuthToggleChoice(state.RemoteAuth.ChoiceIndex)
			return ret(m, nil, true)
		case teakey.Down:
			state.RemoteAuth.ChoiceIndex = remoteAuthToggleChoice(state.RemoteAuth.ChoiceIndex)
			return ret(m, nil, true)
		case teakey.Enter:
			if state.RemoteAuth.ChoiceIndex == 0 {
				state.RemoteAuth.Step = AuthStepPassword
				state.RemoteAuth.Input = textinput.New()
				state.RemoteAuth.Input.Placeholder = i18n.T(i18n.KeyRemoteAuthPasswordPlaceholder)
				state.RemoteAuth.Input.EchoMode = textinput.EchoPassword
				state.RemoteAuth.Input.Focus()
				return ret(m, nil, true)
			}
			state.RemoteAuth.Step = AuthStepIdentity
			initRemoteAuthIdentityInput(&state)
			pcState.Candidates = nil
			pcState.Index = -1
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		case "1":
			state.RemoteAuth.Step = AuthStepPassword
			state.RemoteAuth.Input = textinput.New()
			state.RemoteAuth.Input.Placeholder = i18n.T(i18n.KeyRemoteAuthPasswordPlaceholder)
			state.RemoteAuth.Input.EchoMode = textinput.EchoPassword
			state.RemoteAuth.Input.Focus()
			return ret(m, nil, true)
		case "2":
			state.RemoteAuth.Step = AuthStepIdentity
			initRemoteAuthIdentityInput(&state)
			pcState.Candidates = nil
			pcState.Index = -1
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}
		return ret(m, nil, true)

	case AuthStepPassword:
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}
		if key == teakey.Enter {
			input := state.RemoteAuth.Input.Value()
			if input == "" {
				state.RemoteAuth.Step = AuthStepChoose
				state.RemoteAuth.ChoiceIndex = 0
				return ret(m, nil, true)
			}

			state.RemoteAuth.Connecting = true
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
					Target:   state.RemoteAuth.Target,
					Username: state.RemoteAuth.Username,
					Kind:     remoteauth.ResponseKindPassword,
					Password: input,
				}})
			}
			return ret(m, nil, true)
		}

		var cmd tea.Cmd
		state.RemoteAuth.Input, cmd = state.RemoteAuth.Input.Update(msg)
		return ret(m, cmd, true)

	case AuthStepIdentity:
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}

		cands := pcState.Candidates
		if key == teakey.Up && len(cands) > 0 {
			pcState.Index--
			if pcState.Index < 0 {
				pcState.Index = len(cands) - 1
			}
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}
		if key == teakey.Down && len(cands) > 0 {
			pcState.Index = (pcState.Index + 1) % len(cands)
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}

		pickIdentityCandidate := len(cands) > 0 &&
			pcState.Index >= 0 &&
			pcState.Index < len(cands) &&
			(key == teakey.Enter || key == teakey.Tab)
		if pickIdentityCandidate {
			chosen := cands[pcState.Index]
			state.RemoteAuth.Input.SetValue(chosen)
			state.RemoteAuth.Input.CursorEnd()
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

		if key == teakey.Enter {
			input := state.RemoteAuth.Input.Value()
			if input == "" {
				state.RemoteAuth.Step = AuthStepChoose
				state.RemoteAuth.ChoiceIndex = 0
				pcState.Candidates = nil
				pcState.Index = -1
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
			state.RemoteAuth.Connecting = true
			if m.CommandSender != nil {
				_ = m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: remoteauth.Response{
					Target:   state.RemoteAuth.Target,
					Username: state.RemoteAuth.Username,
					Kind:     remoteauth.ResponseKindIdentity,
					Password: input,
				}})
			}
			return ret(m, nil, true)
		}

		if key == teakey.Tab {
			pcState.Candidates = pathcomplete.Candidates(state.RemoteAuth.Input.Value())
			if len(pcState.Candidates) > 0 {
				pcState.Index = (pcState.Index + 1) % len(pcState.Candidates)
			} else {
				pcState.Index = -1
			}
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}

		var cmd tea.Cmd
		state.RemoteAuth.Input, cmd = state.RemoteAuth.Input.Update(msg)
		pcState.Candidates = pathcomplete.Candidates(state.RemoteAuth.Input.Value())
		if len(pcState.Candidates) > 0 {
			pcState.Index = 0
		} else {
			pcState.Index = -1
		}
		pathcomplete.SetState(pcState)
		return ret(m, cmd, true)
	}

	return ret(m, nil, true)
}

func remoteAuthToggleChoice(choiceIndex int) int {
	if choiceIndex == 1 {
		return 0
	}
	return 1
}
