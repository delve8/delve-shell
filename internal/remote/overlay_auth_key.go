package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/remote/auth"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui"
)

func handleRemoteAuthOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	ret := func(model ui.Model, cmd tea.Cmd, handled bool) (ui.Model, tea.Cmd, bool) {
		setRemoteOverlayState(state)
		return model, cmd, handled
	}

	switch state.RemoteAuth.Step {
	case "hostkey":
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}
		switch key {
		case "1":
			state.RemoteAuth.Connecting = true
			_ = m.EmitRemoteAuthResponseIntent(remoteauth.Response{
				Target: state.RemoteAuth.Target,
				Kind:   "hostkey_accept",
			})
			return ret(m, nil, true)
		case "2":
			_ = m.EmitRemoteAuthResponseIntent(remoteauth.Response{
				Target: state.RemoteAuth.Target,
				Kind:   "hostkey_reject",
			})
			m = m.CloseOverlayVisual()
			state.RemoteAuth = RemoteAuthOverlayState{}
			return ret(m, nil, true)
		}
		return ret(m, nil, true)

	case "auto_identity":
		return ret(m, nil, true)

	case "username":
		if key == teakey.Enter {
			state.RemoteAuth.Username = strings.TrimSpace(state.RemoteAuth.UsernameInput.Value())
			if state.RemoteAuth.Username == "" {
				state.RemoteAuth.Username = "root"
			}
			state.RemoteAuth.Step = "choose"
			return ret(m, nil, true)
		}
		var cmd tea.Cmd
		state.RemoteAuth.UsernameInput, cmd = state.RemoteAuth.UsernameInput.Update(msg)
		return ret(m, cmd, true)

	case "choose":
		switch key {
		case "1":
			state.RemoteAuth.Step = "password"
			state.RemoteAuth.Input = textinput.New()
			state.RemoteAuth.Input.Placeholder = "SSH password"
			state.RemoteAuth.Input.EchoMode = textinput.EchoPassword
			state.RemoteAuth.Input.Focus()
			return ret(m, nil, true)
		case "2":
			state.RemoteAuth.Step = "identity"
			state.RemoteAuth.Input = textinput.New()
			state.RemoteAuth.Input.Placeholder = "~/.ssh/id_rsa"
			state.RemoteAuth.Input.EchoMode = textinput.EchoNormal
			state.RemoteAuth.Input.Focus()
			pcState.Candidates = nil
			pcState.Index = -1
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}
		return ret(m, nil, true)

	case "password":
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}
		if key == teakey.Enter {
			input := state.RemoteAuth.Input.Value()
			if input == "" {
				state.RemoteAuth.Step = "choose"
				m.Interaction.ChoiceIndex = 0
				return ret(m, nil, true)
			}

			state.RemoteAuth.Connecting = true
			_ = m.EmitRemoteAuthResponseIntent(remoteauth.Response{
				Target:   state.RemoteAuth.Target,
				Username: state.RemoteAuth.Username,
				Kind:     state.RemoteAuth.Step,
				Password: input,
			})
			return ret(m, nil, true)
		}

		var cmd tea.Cmd
		state.RemoteAuth.Input, cmd = state.RemoteAuth.Input.Update(msg)
		return ret(m, cmd, true)

	case "identity":
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
				state.RemoteAuth.Step = "choose"
				m.Interaction.ChoiceIndex = 0
				pcState.Candidates = nil
				pcState.Index = -1
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
			state.RemoteAuth.Connecting = true
			_ = m.EmitRemoteAuthResponseIntent(remoteauth.Response{
				Target:   state.RemoteAuth.Target,
				Username: state.RemoteAuth.Username,
				Kind:     state.RemoteAuth.Step,
				Password: input,
			})
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
