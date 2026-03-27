package remote

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/pathcomplete"
	"delve-shell/internal/ui"
)

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
			if err := config.UpdateRemote(target, name, keyPath); err != nil {
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
				ui.SuggestStyleRender(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)),
				"",
			)
			m = m.CloseOverlayVisual()
			state.AddRemote.Active = false
			state.AddRemote.Error = ""
			state.AddRemote.OfferOverwrite = false
			m.Input.Focus()
			m.EmitConfigUpdatedIntent()
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
		if state.AddRemote.Save {
			if err := config.AddRemote(target, name, keyPath); err != nil {
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
				ui.SuggestStyleRender(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)),
				"",
			)
			m.EmitConfigUpdatedIntent()
		}
		if state.AddRemote.Connect {
			state.AddRemote.Connecting = true
			state.AddRemote.Error = ""
			if !m.EmitRemoteOnTargetIntent(target) {
				state.AddRemote.Connecting = false
			}
			return ret(m, nil, true)
		}

		m = m.CloseOverlayVisual()
		state.AddRemote.Active = false
		state.AddRemote.Error = ""
		state.AddRemote.OfferOverwrite = false
		m.Input.Focus()
		return ret(m, nil, true)
	}

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
		cmd = nil
	}
	pathcomplete.SetState(pcState)
	_ = msg
	return ret(m, cmd, true)
}
