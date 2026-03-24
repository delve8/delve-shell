package remote

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/remotesvc"
	"delve-shell/internal/ui"
)

var (
	suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func handleRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	if key == "esc" {
		// Let internal/ui do overlay-close common behavior.
		return m, nil, false
	}
	if m.AddRemote.Active {
		return handleAddRemoteOverlayKey(m, key, msg)
	}
	if m.RemoteAuth.Step != "" {
		return handleRemoteAuthOverlayKey(m, key, msg)
	}
	return m, nil, false
}

func handleAddRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	switch key {
	case "tab":
		if m.AddRemote.FieldIndex == 3 {
			cands := m.PathCompletionCandidates
			if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
				chosen := cands[m.PathCompletionIndex]
				m.AddRemote.KeyInput.SetValue(chosen)
				m.AddRemote.KeyInput.CursorEnd()
				if strings.HasSuffix(chosen, "/") {
					m.PathCompletionCandidates = ui.PathCandidates(chosen)
					m.PathCompletionIndex = 0
				} else {
					m.PathCompletionCandidates = nil
					m.PathCompletionIndex = -1
				}
				return m, nil, true
			}
		}
	case "up", "down":
		// In Key path with completion list: move within list. Else: Up/Down move focus between fields.
		if m.AddRemote.FieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
			cands := m.PathCompletionCandidates
			if key == "up" {
				m.PathCompletionIndex--
				if m.PathCompletionIndex < 0 {
					m.PathCompletionIndex = len(cands) - 1
				}
				return m, nil, true
			}
			if key == "down" {
				m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(cands)
				return m, nil, true
			}
		}

		dir := 1
		if key == "up" {
			dir = -1
		}
		// Field count: 4 for /config add-remote, 5 (with save checkbox) for /remote on.
		fieldCount := 4
		if m.AddRemote.Connect {
			fieldCount = 5
		}
		m.AddRemote.FieldIndex = (m.AddRemote.FieldIndex + dir + fieldCount) % fieldCount
		m.AddRemote.UserInput.Blur()
		m.AddRemote.HostInput.Blur()
		m.AddRemote.NameInput.Blur()
		m.AddRemote.KeyInput.Blur()
		switch m.AddRemote.FieldIndex {
		case 0:
			m.AddRemote.HostInput.Focus()
		case 1:
			m.AddRemote.UserInput.Focus()
		case 2:
			m.AddRemote.NameInput.Focus()
		case 3:
			m.AddRemote.KeyInput.Focus()
		case 4:
			// Save checkbox: no textinput to focus.
		}
		if m.AddRemote.FieldIndex != 3 {
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
		} else {
			m.PathCompletionCandidates = ui.PathCandidates(m.AddRemote.KeyInput.Value())
			if len(m.PathCompletionCandidates) > 0 {
				m.PathCompletionIndex = 0
			} else {
				m.PathCompletionIndex = -1
			}
		}
		return m, nil, true

	case "y", "Y":
		if m.AddRemote.OfferOverwrite {
			host := strings.TrimSpace(m.AddRemote.HostInput.Value())
			user := strings.TrimSpace(m.AddRemote.UserInput.Value())
			if user == "" {
				user = "root"
			}
			name := strings.TrimSpace(m.AddRemote.NameInput.Value())
			keyPath := strings.TrimSpace(m.AddRemote.KeyInput.Value())
			if host == "" {
				return m, nil, true
			}
			target := user + "@" + host
			if err := remotesvc.Update(target, name, keyPath); err != nil {
				m.AddRemote.Error = err.Error()
				m.AddRemote.OfferOverwrite = false
				return m, nil, true
			}
			display := host
			if name != "" {
				display = name + " (" + host + ")"
			}
			lang := "en"
			delvPrefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
			m.Messages = append(m.Messages, suggestStyle.Render(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
			m.Messages = append(m.Messages, "")
			// Refresh content before closing overlay to preserve old behavior.
			m = m.RefreshViewport()

			m.OverlayActive = false
			m.AddRemote.Active = false
			m.AddRemote.Error = ""
			m.AddRemote.OfferOverwrite = false
			m.OverlayTitle = ""
			m.OverlayContent = ""
			m.Input.Focus()
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
			return m, nil, true
		}

	case " ":
		// Space toggles save-as-remote only when focused on the checkbox field.
		if m.AddRemote.FieldIndex == 4 {
			m.AddRemote.Save = !m.AddRemote.Save
			return m, nil, true
		}

	case "enter":
		if m.AddRemote.FieldIndex == 3 {
			cands := m.PathCompletionCandidates
			if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
				chosen := cands[m.PathCompletionIndex]
				m.AddRemote.KeyInput.SetValue(chosen)
				m.AddRemote.KeyInput.CursorEnd()
				if strings.HasSuffix(chosen, "/") {
					m.PathCompletionCandidates = ui.PathCandidates(chosen)
					m.PathCompletionIndex = 0
				} else {
					m.PathCompletionCandidates = nil
					m.PathCompletionIndex = -1
				}
				return m, nil, true
			}
		}

		host := strings.TrimSpace(m.AddRemote.HostInput.Value())
		user := strings.TrimSpace(m.AddRemote.UserInput.Value())
		if user == "" {
			user = "root"
		}
		name := strings.TrimSpace(m.AddRemote.NameInput.Value())
		keyPath := strings.TrimSpace(m.AddRemote.KeyInput.Value())

		if host == "" {
			m.AddRemote.Error = "host is required"
			return m, nil, true
		}
		if strings.Contains(host, "@") {
			m.AddRemote.Error = "host must not contain @"
			return m, nil, true
		}

		target := user + "@" + host
		// Optionally save/update remote config when requested.
		if m.AddRemote.Save {
			if err := remotesvc.Add(target, name, keyPath); err != nil {
				m.AddRemote.Error = err.Error()
				m.AddRemote.OfferOverwrite = strings.Contains(err.Error(), "already exists")
				return m, nil, true
			}
			m.AddRemote.OfferOverwrite = false
			display := host
			if name != "" {
				display = name + " (" + host + ")"
			}
			lang := "en"
			delvPrefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
			m.Messages = append(m.Messages, suggestStyle.Render(delvPrefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
			m.Messages = append(m.Messages, "")
			if m.ConfigUpdatedChan != nil {
				select {
				case m.ConfigUpdatedChan <- struct{}{}:
				default:
				}
			}
		}

		m = m.RefreshViewport()
		if m.AddRemote.Connect && m.RemoteOnChan != nil {
			// Show "Connecting..." and wait for RemoteConnectDoneMsg; close overlay only on success.
			m.AddRemote.Connecting = true
			m.AddRemote.Error = ""
			select {
			case m.RemoteOnChan <- target:
			default:
				m.AddRemote.Connecting = false
			}
			return m, nil, true
		}

		m.OverlayActive = false
		m.AddRemote.Active = false
		m.AddRemote.Error = ""
		m.AddRemote.OfferOverwrite = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		m.Input.Focus()
		return m, nil, true
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch m.AddRemote.FieldIndex {
	case 0:
		m.AddRemote.HostInput, cmd = m.AddRemote.HostInput.Update(msg)
	case 1:
		m.AddRemote.UserInput, cmd = m.AddRemote.UserInput.Update(msg)
	case 2:
		m.AddRemote.NameInput, cmd = m.AddRemote.NameInput.Update(msg)
	case 3:
		m.AddRemote.KeyInput, cmd = m.AddRemote.KeyInput.Update(msg)
		m.PathCompletionCandidates = ui.PathCandidates(m.AddRemote.KeyInput.Value())
		if len(m.PathCompletionCandidates) > 0 {
			m.PathCompletionIndex = 0
		} else {
			m.PathCompletionIndex = -1
		}
	case 4:
		// Save checkbox has no text input; ignore character keys.
		cmd = nil
	}
	_ = msg
	return m, cmd, true
}

func handleRemoteAuthOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	// Keep step specific behavior identical to internal/ui's prior switch.
	switch m.RemoteAuth.Step {
	case "auto_identity":
		// No interactive input; Esc handled by ui.
		return m, nil, true
	case "username":
		if key == "enter" {
			m.RemoteAuth.Username = strings.TrimSpace(m.RemoteAuth.UsernameInput.Value())
			if m.RemoteAuth.Username == "" {
				m.RemoteAuth.Username = "root"
			}
			m.RemoteAuth.Step = "choose"
			return m, nil, true
		}
		var cmd tea.Cmd
		m.RemoteAuth.UsernameInput, cmd = m.RemoteAuth.UsernameInput.Update(msg)
		return m, cmd, true
	case "choose":
		switch key {
		case "1":
			m.RemoteAuth.Step = "password"
			m.RemoteAuth.Input = textinput.New()
			m.RemoteAuth.Input.Placeholder = "SSH password"
			m.RemoteAuth.Input.EchoMode = textinput.EchoPassword
			m.RemoteAuth.Input.Focus()
			var b strings.Builder
			if m.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.OverlayContent = b.String()
			return m, nil, true
		case "2":
			m.RemoteAuth.Step = "identity"
			m.RemoteAuth.Input = textinput.New()
			m.RemoteAuth.Input.Placeholder = "~/.ssh/id_rsa"
			m.RemoteAuth.Input.EchoMode = textinput.EchoNormal
			m.RemoteAuth.Input.Focus()
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
			var b strings.Builder
			if m.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.OverlayContent = b.String()
			return m, nil, true
		}
		return m, nil, true
	case "password":
		// When waiting for auth result, ignore further input except Esc (handled above).
		if m.RemoteAuth.Connecting {
			return m, nil, true
		}
		if key == "enter" {
			input := m.RemoteAuth.Input.Value()
			if input == "" {
				m.RemoteAuth.Step = "choose"
				m.ChoiceIndex = 0
				var b strings.Builder
				if m.RemoteAuth.Error != "" {
					b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.OverlayContent = b.String()
				return m, nil, true
			}

			// Non-empty password: show connecting state and send credentials.
			m.RemoteAuth.Connecting = true
			var b strings.Builder
			if m.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.OverlayContent = b.String()
			if m.RemoteAuthRespChan != nil {
				select {
				case m.RemoteAuthRespChan <- ui.RemoteAuthResponse{
					Target:   m.RemoteAuth.Target,
					Username: m.RemoteAuth.Username,
					Kind:     m.RemoteAuth.Step,
					Password: input,
				}:
				default:
				}
			}
			return m, nil, true
		}

		var cmd tea.Cmd
		m.RemoteAuth.Input, cmd = m.RemoteAuth.Input.Update(msg)
		return m, cmd, true

	case "identity":
		// When waiting for auth result, ignore further input except Esc (handled above).
		if m.RemoteAuth.Connecting {
			return m, nil, true
		}

		cands := m.PathCompletionCandidates
		if key == "up" && len(cands) > 0 {
			m.PathCompletionIndex--
			if m.PathCompletionIndex < 0 {
				m.PathCompletionIndex = len(cands) - 1
			}
			return m, nil, true
		}
		if key == "down" && len(cands) > 0 {
			m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(cands)
			return m, nil, true
		}

		pickIdentityCandidate := len(cands) > 0 &&
			m.PathCompletionIndex >= 0 &&
			m.PathCompletionIndex < len(cands) &&
			(key == "enter" || key == "tab")
		if pickIdentityCandidate {
			chosen := cands[m.PathCompletionIndex]
			m.RemoteAuth.Input.SetValue(chosen)
			m.RemoteAuth.Input.CursorEnd()
			if strings.HasSuffix(chosen, "/") {
				m.PathCompletionCandidates = ui.PathCandidates(chosen)
				m.PathCompletionIndex = 0
			} else {
				m.PathCompletionCandidates = nil
				m.PathCompletionIndex = -1
			}
			return m, nil, true
		}

		if key == "enter" {
			input := m.RemoteAuth.Input.Value()
			if input == "" {
				m.RemoteAuth.Step = "choose"
				m.ChoiceIndex = 0
				m.PathCompletionCandidates = nil
				m.PathCompletionIndex = -1
				var b strings.Builder
				if m.RemoteAuth.Error != "" {
					b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.OverlayContent = b.String()
				return m, nil, true
			}
			m.RemoteAuth.Connecting = true
			var b strings.Builder
			if m.RemoteAuth.Error != "" {
				b.WriteString(errStyle.Render(m.RemoteAuth.Error) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuth.Target) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.OverlayContent = b.String()
			if m.RemoteAuthRespChan != nil {
				select {
				case m.RemoteAuthRespChan <- ui.RemoteAuthResponse{
					Target:   m.RemoteAuth.Target,
					Username: m.RemoteAuth.Username,
					Kind:     m.RemoteAuth.Step,
					Password: input,
				}:
				default:
				}
			}
			return m, nil, true
		}

		if key == "tab" {
			m.PathCompletionCandidates = ui.PathCandidates(m.RemoteAuth.Input.Value())
			if len(m.PathCompletionCandidates) > 0 {
				m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(m.PathCompletionCandidates)
			} else {
				m.PathCompletionIndex = -1
			}
			return m, nil, true
		}

		var cmd tea.Cmd
		m.RemoteAuth.Input, cmd = m.RemoteAuth.Input.Update(msg)
		// Refresh path candidates from new input (so dropdown updates as user types).
		m.PathCompletionCandidates = ui.PathCandidates(m.RemoteAuth.Input.Value())
		if len(m.PathCompletionCandidates) > 0 {
			m.PathCompletionIndex = 0
		} else {
			m.PathCompletionIndex = -1
		}
		return m, cmd, true
	}

	return m, nil, true
}
