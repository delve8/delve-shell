package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/service/remotesvc"
	"delve-shell/internal/ui"
)

var (
	suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func handleRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	if key == "esc" {
		// Let internal/ui do overlay-close common behavior.
		return m, nil, false
	}
	if state.AddRemote.Active {
		return handleAddRemoteOverlayKey(m, key, msg)
	}
	if state.RemoteAuth.Step != "" {
		return handleRemoteAuthOverlayKey(m, key, msg)
	}
	return m, nil, false
}

func handleAddRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	ret := func(model ui.Model, cmd tea.Cmd, handled bool) (ui.Model, tea.Cmd, bool) {
		setRemoteOverlayState(state)
		return model, cmd, handled
	}
	switch key {
	case "tab":
		if state.AddRemote.FieldIndex == 3 {
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
	case "up", "down":
		// In Key path with completion list: move within list. Else: Up/Down move focus between fields.
		if state.AddRemote.FieldIndex == 3 && len(pcState.Candidates) > 0 {
			cands := pcState.Candidates
			if key == "up" {
				pcState.Index--
				if pcState.Index < 0 {
					pcState.Index = len(cands) - 1
				}
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
			if key == "down" {
				pcState.Index = (pcState.Index + 1) % len(cands)
				pathcomplete.SetState(pcState)
				return ret(m, nil, true)
			}
		}

		dir := 1
		if key == "up" {
			dir = -1
		}
		// Field count: 4 for /config add-remote, 5 (with save checkbox) for /remote on.
		fieldCount := 4
		if state.AddRemote.Connect {
			fieldCount = 5
		}
		state.AddRemote.FieldIndex = (state.AddRemote.FieldIndex + dir + fieldCount) % fieldCount
		state.AddRemote.UserInput.Blur()
		state.AddRemote.HostInput.Blur()
		state.AddRemote.NameInput.Blur()
		state.AddRemote.KeyInput.Blur()
		switch state.AddRemote.FieldIndex {
		case 0:
			state.AddRemote.HostInput.Focus()
		case 1:
			state.AddRemote.UserInput.Focus()
		case 2:
			state.AddRemote.NameInput.Focus()
		case 3:
			state.AddRemote.KeyInput.Focus()
		case 4:
			// Save checkbox: no textinput to focus.
		}
		if state.AddRemote.FieldIndex != 3 {
			pcState.Candidates = nil
			pcState.Index = -1
		} else {
			pcState.Candidates = pathcomplete.Candidates(state.AddRemote.KeyInput.Value())
			if len(pcState.Candidates) > 0 {
				pcState.Index = 0
			} else {
				pcState.Index = -1
			}
		}
		pathcomplete.SetState(pcState)
		return ret(m, nil, true)

	case "y", "Y":
		if state.AddRemote.OfferOverwrite {
			host := strings.TrimSpace(state.AddRemote.HostInput.Value())
			user := strings.TrimSpace(state.AddRemote.UserInput.Value())
			if user == "" {
				user = "root"
			}
			name := strings.TrimSpace(state.AddRemote.NameInput.Value())
			keyPath := strings.TrimSpace(state.AddRemote.KeyInput.Value())
			if host == "" {
				return ret(m, nil, true)
			}
			target := user + "@" + host
			if err := remotesvc.Update(target, name, keyPath); err != nil {
				state.AddRemote.Error = err.Error()
				state.AddRemote.OfferOverwrite = false
				return ret(m, nil, true)
			}
			display := host
			if name != "" {
				display = name + " (" + host + ")"
			}
			lang := "en"
			delvPrefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
			m = m.AppendTranscriptLines(
				suggestStyle.Render(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)),
				"",
			)
			// Refresh content before closing overlay to preserve old behavior.
			m = m.RefreshViewport()

			m.Overlay.Active = false
			state.AddRemote.Active = false
			state.AddRemote.Error = ""
			state.AddRemote.OfferOverwrite = false
			m.Overlay.Title = ""
			m.Overlay.Content = ""
			m.Input.Focus()
			m.NotifyConfigUpdated()
			return ret(m, nil, true)
		}

	case " ":
		// Space toggles save-as-remote only when focused on the checkbox field.
		if state.AddRemote.FieldIndex == 4 {
			state.AddRemote.Save = !state.AddRemote.Save
			return ret(m, nil, true)
		}

	case "enter":
		if state.AddRemote.FieldIndex == 3 {
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

		host := strings.TrimSpace(state.AddRemote.HostInput.Value())
		user := strings.TrimSpace(state.AddRemote.UserInput.Value())
		if user == "" {
			user = "root"
		}
		name := strings.TrimSpace(state.AddRemote.NameInput.Value())
		keyPath := strings.TrimSpace(state.AddRemote.KeyInput.Value())

		if host == "" {
			state.AddRemote.Error = "host is required"
			return ret(m, nil, true)
		}
		if strings.Contains(host, "@") {
			state.AddRemote.Error = "host must not contain @"
			return ret(m, nil, true)
		}

		target := user + "@" + host
		// Optionally save/update remote config when requested.
		if state.AddRemote.Save {
			if err := remotesvc.Add(target, name, keyPath); err != nil {
				state.AddRemote.Error = err.Error()
				state.AddRemote.OfferOverwrite = strings.Contains(err.Error(), "already exists")
				return ret(m, nil, true)
			}
			state.AddRemote.OfferOverwrite = false
			display := host
			if name != "" {
				display = name + " (" + host + ")"
			}
			lang := "en"
			delvPrefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
			m = m.AppendTranscriptLines(
				suggestStyle.Render(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)),
				"",
			)
			m.NotifyConfigUpdated()
		}

		m = m.RefreshViewport()
		if state.AddRemote.Connect {
			// Show "Connecting..." and wait for RemoteConnectDoneMsg; close overlay only on success.
			state.AddRemote.Connecting = true
			state.AddRemote.Error = ""
			if !m.PublishRemoteOnTarget(target) {
				state.AddRemote.Connecting = false
			}
			return ret(m, nil, true)
		}

		m.Overlay.Active = false
		state.AddRemote.Active = false
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
		m.Overlay.Title = ""
		m.Overlay.Content = ""
		m.Input.Focus()
		return ret(m, nil, true)
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch state.AddRemote.FieldIndex {
	case 0:
		state.AddRemote.HostInput, cmd = state.AddRemote.HostInput.Update(msg)
	case 1:
		state.AddRemote.UserInput, cmd = state.AddRemote.UserInput.Update(msg)
	case 2:
		state.AddRemote.NameInput, cmd = state.AddRemote.NameInput.Update(msg)
	case 3:
		state.AddRemote.KeyInput, cmd = state.AddRemote.KeyInput.Update(msg)
		pcState.Candidates = pathcomplete.Candidates(state.AddRemote.KeyInput.Value())
		if len(pcState.Candidates) > 0 {
			pcState.Index = 0
		} else {
			pcState.Index = -1
		}
	case 4:
		// Save checkbox has no text input; ignore character keys.
		cmd = nil
	}
	pathcomplete.SetState(pcState)
	_ = msg
	return ret(m, cmd, true)
}

func handleRemoteAuthOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	state := getRemoteOverlayState()
	pcState := pathcomplete.GetState()
	ret := func(model ui.Model, cmd tea.Cmd, handled bool) (ui.Model, tea.Cmd, bool) {
		setRemoteOverlayState(state)
		return model, cmd, handled
	}
	// Keep step specific behavior identical to internal/ui's prior switch.
	switch state.RemoteAuth.Step {
	case "auto_identity":
		// No interactive input; Esc handled by ui.
		return ret(m, nil, true)
	case "username":
		if key == "enter" {
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
			var b strings.Builder
			if state.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(state.RemoteAuth.Target) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.Overlay.Content = b.String()
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
			var b strings.Builder
			if state.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(state.RemoteAuth.Target) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.Overlay.Content = b.String()
			return ret(m, nil, true)
		}
		return ret(m, nil, true)
	case "password":
		// When waiting for auth result, ignore further input except Esc (handled above).
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}
		if key == "enter" {
			input := state.RemoteAuth.Input.Value()
			if input == "" {
				state.RemoteAuth.Step = "choose"
				m.Interaction.ChoiceIndex = 0
				var b strings.Builder
				if state.RemoteAuth.Error != "" {
					b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.Overlay.Content = b.String()
				return ret(m, nil, true)
			}

			// Non-empty password: show connecting state and send credentials.
			state.RemoteAuth.Connecting = true
			var b strings.Builder
			if state.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(state.RemoteAuth.Target) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.Overlay.Content = b.String()
			_ = m.PublishRemoteAuthResponse(remoteauth.Response{
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
		// When waiting for auth result, ignore further input except Esc (handled above).
		if state.RemoteAuth.Connecting {
			return ret(m, nil, true)
		}

		cands := pcState.Candidates
		if key == "up" && len(cands) > 0 {
			pcState.Index--
			if pcState.Index < 0 {
				pcState.Index = len(cands) - 1
			}
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}
		if key == "down" && len(cands) > 0 {
			pcState.Index = (pcState.Index + 1) % len(cands)
			pathcomplete.SetState(pcState)
			return ret(m, nil, true)
		}

		pickIdentityCandidate := len(cands) > 0 &&
			pcState.Index >= 0 &&
			pcState.Index < len(cands) &&
			(key == "enter" || key == "tab")
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

		if key == "enter" {
			input := state.RemoteAuth.Input.Value()
			if input == "" {
				state.RemoteAuth.Step = "choose"
				m.Interaction.ChoiceIndex = 0
				pcState.Candidates = nil
				pcState.Index = -1
				pathcomplete.SetState(pcState)
				var b strings.Builder
				if state.RemoteAuth.Error != "" {
					b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.Overlay.Content = b.String()
				return ret(m, nil, true)
			}
			state.RemoteAuth.Connecting = true
			var b strings.Builder
			if state.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(state.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(state.RemoteAuth.Target) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.Overlay.Content = b.String()
			_ = m.PublishRemoteAuthResponse(remoteauth.Response{
				Target:   state.RemoteAuth.Target,
				Username: state.RemoteAuth.Username,
				Kind:     state.RemoteAuth.Step,
				Password: input,
			})
			return ret(m, nil, true)
		}

		if key == "tab" {
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
		// Refresh path candidates from new input (so dropdown updates as user types).
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
