package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/service/remotesvc"
)

// handleOverlayKey routes key input when overlay is active.
func (m Model) handleOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}

	switch key {
	case "esc":
		m, cmd := m.closeOverlayCommon(true)
		return m, cmd, true
	default:
		// Add-skill overlay: URL, ref, path.
		if m2, cmd, handled := m.handleAddSkillOverlayKey(key, msg); handled {
			return m2, cmd, true
		}
		// Add-remote overlay: form with 5 fields (host, username, name, key path, save-as-remote checkbox).
		if m.AddRemoteActive {
			switch key {
			case "tab":
				// Tab only selects path candidate when list is visible; no longer used to move between fields.
				if m.AddRemoteFieldIndex == 3 {
					cands := m.PathCompletionCandidates
					if len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) {
						chosen := cands[m.PathCompletionIndex]
						m.AddRemoteKeyInput.SetValue(chosen)
						m.AddRemoteKeyInput.CursorEnd()
						if strings.HasSuffix(chosen, "/") {
							m.PathCompletionCandidates = PathCandidates(chosen)
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
					m.PathCompletionCandidates = PathCandidates(m.AddRemoteKeyInput.Value())
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
					lang := m.getLang()
					m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display))))
					m.Messages = append(m.Messages, "")
					m.Viewport.SetContent(m.buildContent())
					m.Viewport.GotoBottom()
					m.OverlayActive = false
					m.AddRemoteActive = false
					m.AddRemoteError = ""
					m.AddRemoteOfferOverwrite = false
					m.OverlayTitle = ""
					m.OverlayContent = ""
					// After closing Add Remote overlay (overwrite), refocus main input.
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
							m.PathCompletionCandidates = PathCandidates(chosen)
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
					lang := m.getLang()
					m.Messages = append(m.Messages, suggestStyle.Render(m.delveMsg(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display))))
					m.Messages = append(m.Messages, "")
					if m.ConfigUpdatedChan != nil {
						select {
						case m.ConfigUpdatedChan <- struct{}{}:
						default:
						}
					}
				}
				m.Viewport.SetContent(m.buildContent())
				m.Viewport.GotoBottom()
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
				m.PathCompletionCandidates = PathCandidates(m.AddRemoteKeyInput.Value())
				if len(m.PathCompletionCandidates) > 0 {
					m.PathCompletionIndex = 0
				} else {
					m.PathCompletionIndex = -1
				}
			case 4:
				// Save checkbox has no text input; ignore character keys here.
				cmd = nil
			}
			return m, cmd, true
		}
		if m.ConfigLLMActive {
			const configLLMFieldCount = 5
			switch key {
			case "up", "down":
				dir := 1
				if key == "up" {
					dir = -1
				}
				m.ConfigLLMFieldIndex = (m.ConfigLLMFieldIndex + dir + configLLMFieldCount) % configLLMFieldCount
				m.ConfigLLMBaseURLInput.Blur()
				m.ConfigLLMApiKeyInput.Blur()
				m.ConfigLLMModelInput.Blur()
				m.ConfigLLMMaxMessagesInput.Blur()
				m.ConfigLLMMaxCharsInput.Blur()
				switch m.ConfigLLMFieldIndex {
				case 0:
					m.ConfigLLMBaseURLInput.Focus()
				case 1:
					m.ConfigLLMApiKeyInput.Focus()
				case 2:
					m.ConfigLLMModelInput.Focus()
				case 3:
					m.ConfigLLMMaxMessagesInput.Focus()
				case 4:
					m.ConfigLLMMaxCharsInput.Focus()
				}
				return m, nil, true
			case "enter":
				if m.ConfigLLMChecking {
					return m, nil, true
				}
				baseURL := strings.TrimSpace(m.ConfigLLMBaseURLInput.Value())
				apiKey := strings.TrimSpace(m.ConfigLLMApiKeyInput.Value())
				model := strings.TrimSpace(m.ConfigLLMModelInput.Value())
				maxMessagesStr := strings.TrimSpace(m.ConfigLLMMaxMessagesInput.Value())
				maxCharsStr := strings.TrimSpace(m.ConfigLLMMaxCharsInput.Value())
				if model == "" {
					m.ConfigLLMError = i18n.T(m.getLang(), i18n.KeyConfigLLMModelRequired)
					return m, nil, true
				}
				m = m.applyConfigLLMFromOverlayStart(baseURL, apiKey, model, maxMessagesStr, maxCharsStr)
				if !m.ConfigLLMChecking {
					return m, nil, true
				}
				return m, RunConfigLLMCheckCmd(), true
			}
			var cmd tea.Cmd
			switch m.ConfigLLMFieldIndex {
			case 0:
				m.ConfigLLMBaseURLInput, cmd = m.ConfigLLMBaseURLInput.Update(msg)
			case 1:
				m.ConfigLLMApiKeyInput, cmd = m.ConfigLLMApiKeyInput.Update(msg)
			case 2:
				m.ConfigLLMModelInput, cmd = m.ConfigLLMModelInput.Update(msg)
			case 3:
				m.ConfigLLMMaxMessagesInput, cmd = m.ConfigLLMMaxMessagesInput.Update(msg)
			case 4:
				m.ConfigLLMMaxCharsInput, cmd = m.ConfigLLMMaxCharsInput.Update(msg)
			}
			return m, cmd, true
		}
		// Update-skill overlay: choose ref and confirm update.
		if m2, cmd, handled := m.handleUpdateSkillOverlayKey(key); handled {
			return m2, cmd, true
		}
		// Remote auth: step "username" -> "choose" (1/2) -> "password" or "identity".
		switch m.RemoteAuthStep {
		case "auto_identity":
			// Automatic connection with configured identity file: no interactive input; Esc handled above.
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
				// Non-empty password: show connecting state and send credentials; overlay stays open until auth result.
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
					case m.RemoteAuthRespChan <- RemoteAuthResponse{
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
			// Path completion: Up/Down to move, Enter or Tab to pick candidate (Tab matches bash habit), or submit with Enter.
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
			pickIdentityCandidate := len(cands) > 0 && m.PathCompletionIndex >= 0 && m.PathCompletionIndex < len(cands) && (key == "enter" || key == "tab")
			if pickIdentityCandidate {
				chosen := cands[m.PathCompletionIndex]
				m.RemoteAuthInput.SetValue(chosen)
				m.RemoteAuthInput.CursorEnd()
				if strings.HasSuffix(chosen, "/") {
					m.PathCompletionCandidates = PathCandidates(chosen)
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
				// Non-empty key path: show connecting state and send credentials; overlay stays open until auth result.
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
					case m.RemoteAuthRespChan <- RemoteAuthResponse{
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
				// No candidate selected: refresh list (Tab already handled pick above when candidates exist).
				m.PathCompletionCandidates = PathCandidates(m.RemoteAuthInput.Value())
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
			m.PathCompletionCandidates = PathCandidates(m.RemoteAuthInput.Value())
			if len(m.PathCompletionCandidates) > 0 {
				m.PathCompletionIndex = 0
			} else {
				m.PathCompletionIndex = -1
			}
			return m, cmd, true
		}
		// Generic overlay: pass up/down/pgup/pgdown.
		var cmd tea.Cmd
		m.OverlayViewport, cmd = m.OverlayViewport.Update(msg)
		return m, cmd, true
	}
}
