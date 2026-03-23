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
	if m.AddRemoteActive {
		return handleAddRemoteOverlayKey(m, key, msg)
	}
	if m.RemoteAuthStep != "" {
		return handleRemoteAuthOverlayKey(m, key, msg)
	}
	return m, nil, false
}

func handleAddRemoteOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	switch key {
	case "tab":
		if m.AddRemoteFieldIndex == 3 {
			cands := m.PathCompletionCandidates
			if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
				chosen := cands[m.PathCompletionIndex]
				m.AddRemoteKeyInput.SetValue(chosen)
				m.AddRemoteKeyInput.CursorEnd()
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
		if m.AddRemoteFieldIndex == 3 && len(m.PathCompletionCandidates) > 0 {
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
		if m.AddRemoteConnect {
			fieldCount = 5
		}
		m.AddRemoteFieldIndex = (m.AddRemoteFieldIndex + dir + fieldCount) % fieldCount
		m.AddRemoteUserInput.Blur()
		m.AddRemoteHostInput.Blur()
		m.AddRemoteNameInput.Blur()
		m.AddRemoteKeyInput.Blur()
		switch m.AddRemoteFieldIndex {
		case 0:
			m.AddRemoteHostInput.Focus()
		case 1:
			m.AddRemoteUserInput.Focus()
		case 2:
			m.AddRemoteNameInput.Focus()
		case 3:
			m.AddRemoteKeyInput.Focus()
		case 4:
			// Save checkbox: no textinput to focus.
		}
		if m.AddRemoteFieldIndex != 3 {
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
		} else {
			m.PathCompletionCandidates = ui.PathCandidates(m.AddRemoteKeyInput.Value())
			if len(m.PathCompletionCandidates) > 0 {
				m.PathCompletionIndex = 0
			} else {
				m.PathCompletionIndex = -1
			}
		}
		return m, nil, true

	case "y", "Y":
		if m.AddRemoteOfferOverwrite {
			host := strings.TrimSpace(m.AddRemoteHostInput.Value())
			user := strings.TrimSpace(m.AddRemoteUserInput.Value())
			if user == "" {
				user = "root"
			}
			name := strings.TrimSpace(m.AddRemoteNameInput.Value())
			keyPath := strings.TrimSpace(m.AddRemoteKeyInput.Value())
			if host == "" {
				return m, nil, true
			}
			target := user + "@" + host
			if err := remotesvc.Update(target, name, keyPath); err != nil {
				m.AddRemoteError = err.Error()
				m.AddRemoteOfferOverwrite = false
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
			m.AddRemoteActive = false
			m.AddRemoteError = ""
			m.AddRemoteOfferOverwrite = false
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
		if m.AddRemoteFieldIndex == 4 {
			m.AddRemoteSave = !m.AddRemoteSave
			return m, nil, true
		}

	case "enter":
		if m.AddRemoteFieldIndex == 3 {
			cands := m.PathCompletionCandidates
			if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
				chosen := cands[m.PathCompletionIndex]
				m.AddRemoteKeyInput.SetValue(chosen)
				m.AddRemoteKeyInput.CursorEnd()
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

		host := strings.TrimSpace(m.AddRemoteHostInput.Value())
		user := strings.TrimSpace(m.AddRemoteUserInput.Value())
		if user == "" {
			user = "root"
		}
		name := strings.TrimSpace(m.AddRemoteNameInput.Value())
		keyPath := strings.TrimSpace(m.AddRemoteKeyInput.Value())

		if host == "" {
			m.AddRemoteError = "host is required"
			return m, nil, true
		}
		if strings.Contains(host, "@") {
			m.AddRemoteError = "host must not contain @"
			return m, nil, true
		}

		target := user + "@" + host
		// Optionally save/update remote config when requested.
		if m.AddRemoteSave {
			if err := remotesvc.Add(target, name, keyPath); err != nil {
				m.AddRemoteError = err.Error()
				m.AddRemoteOfferOverwrite = strings.Contains(err.Error(), "already exists")
				return m, nil, true
			}
			m.AddRemoteOfferOverwrite = false
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
		if m.AddRemoteConnect && m.RemoteOnChan != nil {
			// Show "Connecting..." and wait for RemoteConnectDoneMsg; close overlay only on success.
			m.AddRemoteConnecting = true
			m.AddRemoteError = ""
			select {
			case m.RemoteOnChan <- target:
			default:
				m.AddRemoteConnecting = false
			}
			return m, nil, true
		}

		m.OverlayActive = false
		m.AddRemoteActive = false
		m.AddRemoteError = ""
		m.AddRemoteOfferOverwrite = false
		m.OverlayTitle = ""
		m.OverlayContent = ""
		m.Input.Focus()
		return m, nil, true
	}

	// Default: forward to active field input.
	var cmd tea.Cmd
	switch m.AddRemoteFieldIndex {
	case 0:
		m.AddRemoteHostInput, cmd = m.AddRemoteHostInput.Update(msg)
	case 1:
		m.AddRemoteUserInput, cmd = m.AddRemoteUserInput.Update(msg)
	case 2:
		m.AddRemoteNameInput, cmd = m.AddRemoteNameInput.Update(msg)
	case 3:
		m.AddRemoteKeyInput, cmd = m.AddRemoteKeyInput.Update(msg)
		m.PathCompletionCandidates = ui.PathCandidates(m.AddRemoteKeyInput.Value())
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
	switch m.RemoteAuthStep {
	case "auto_identity":
		// No interactive input; Esc handled by ui.
		return m, nil, true
	case "username":
		if key == "enter" {
			m.RemoteAuthUsername = strings.TrimSpace(m.RemoteAuthUsernameInput.Value())
			if m.RemoteAuthUsername == "" {
				m.RemoteAuthUsername = "root"
			}
			m.RemoteAuthStep = "choose"
			return m, nil, true
		}
		var cmd tea.Cmd
		m.RemoteAuthUsernameInput, cmd = m.RemoteAuthUsernameInput.Update(msg)
		return m, cmd, true
	case "choose":
		switch key {
		case "1":
			m.RemoteAuthStep = "password"
			m.RemoteAuthInput = textinput.New()
			m.RemoteAuthInput.Placeholder = "SSH password"
			m.RemoteAuthInput.EchoMode = textinput.EchoPassword
			m.RemoteAuthInput.Focus()
			var b strings.Builder
			if m.RemoteAuthError != "" {
				b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.OverlayContent = b.String()
			return m, nil, true
		case "2":
			m.RemoteAuthStep = "identity"
			m.RemoteAuthInput = textinput.New()
			m.RemoteAuthInput.Placeholder = "~/.ssh/id_rsa"
			m.RemoteAuthInput.EchoMode = textinput.EchoNormal
			m.RemoteAuthInput.Focus()
			m.PathCompletionCandidates = nil
			m.PathCompletionIndex = -1
			var b strings.Builder
			if m.RemoteAuthError != "" {
				b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
			b.WriteString("Press Enter to submit, Esc to cancel.")
			m.OverlayContent = b.String()
			return m, nil, true
		}
		return m, nil, true
	case "password":
		// When waiting for auth result, ignore further input except Esc (handled above).
		if m.RemoteAuthConnecting {
			return m, nil, true
		}
		if key == "enter" {
			input := m.RemoteAuthInput.Value()
			if input == "" {
				m.RemoteAuthStep = "choose"
				m.ChoiceIndex = 0
				var b strings.Builder
				if m.RemoteAuthError != "" {
					b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.OverlayContent = b.String()
				return m, nil, true
			}

			// Non-empty password: show connecting state and send credentials.
			m.RemoteAuthConnecting = true
			var b strings.Builder
			if m.RemoteAuthError != "" {
				b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
			}
			b.WriteString("SSH password for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.OverlayContent = b.String()
			if m.RemoteAuthRespChan != nil {
				select {
				case m.RemoteAuthRespChan <- ui.RemoteAuthResponse{
					Target:   m.RemoteAuthTarget,
					Username: m.RemoteAuthUsername,
					Kind:     m.RemoteAuthStep,
					Password: input,
				}:
				default:
				}
			}
			return m, nil, true
		}

		var cmd tea.Cmd
		m.RemoteAuthInput, cmd = m.RemoteAuthInput.Update(msg)
		return m, cmd, true

	case "identity":
		// When waiting for auth result, ignore further input except Esc (handled above).
		if m.RemoteAuthConnecting {
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
			m.RemoteAuthInput.SetValue(chosen)
			m.RemoteAuthInput.CursorEnd()
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
			input := m.RemoteAuthInput.Value()
			if input == "" {
				m.RemoteAuthStep = "choose"
				m.ChoiceIndex = 0
				m.PathCompletionCandidates = nil
				m.PathCompletionIndex = -1
				var b strings.Builder
				if m.RemoteAuthError != "" {
					b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
				}
				b.WriteString("Choose authentication method:\n")
				b.WriteString("  1. Password\n")
				b.WriteString("  2. Key file (identity file)\n\n")
				b.WriteString("Press 1 or 2 to select, Esc to cancel.")
				m.OverlayContent = b.String()
				return m, nil, true
			}
			m.RemoteAuthConnecting = true
			var b strings.Builder
			if m.RemoteAuthError != "" {
				b.WriteString(errStyle.Render(m.RemoteAuthError) + "\n\n")
			}
			b.WriteString("SSH key file path for " + config.HostFromTarget(m.RemoteAuthTarget) + "\n")
			b.WriteString(suggestStyle.Render("Connecting...") + "\n\n")
			b.WriteString("Press Esc to cancel.")
			m.OverlayContent = b.String()
			if m.RemoteAuthRespChan != nil {
				select {
				case m.RemoteAuthRespChan <- ui.RemoteAuthResponse{
					Target:   m.RemoteAuthTarget,
					Username: m.RemoteAuthUsername,
					Kind:     m.RemoteAuthStep,
					Password: input,
				}:
				default:
				}
			}
			return m, nil, true
		}

		if key == "tab" {
			m.PathCompletionCandidates = ui.PathCandidates(m.RemoteAuthInput.Value())
			if len(m.PathCompletionCandidates) > 0 {
				m.PathCompletionIndex = (m.PathCompletionIndex + 1) % len(m.PathCompletionCandidates)
			} else {
				m.PathCompletionIndex = -1
			}
			return m, nil, true
		}

		var cmd tea.Cmd
		m.RemoteAuthInput, cmd = m.RemoteAuthInput.Update(msg)
		// Refresh path candidates from new input (so dropdown updates as user types).
		m.PathCompletionCandidates = ui.PathCandidates(m.RemoteAuthInput.Value())
		if len(m.PathCompletionCandidates) > 0 {
			m.PathCompletionIndex = 0
		} else {
			m.PathCompletionIndex = -1
		}
		return m, cmd, true
	}

	return m, nil, true
}
