package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) closeOverlayCommon(refocusInput bool) (Model, tea.Cmd) {
	m.OverlayActive = false
	m.OverlayTitle = ""
	m.OverlayContent = ""
	m.AddRemoteActive = false
	m.AddRemoteConnecting = false
	m.AddRemoteError = ""
	m.AddRemoteOfferOverwrite = false
	m.RemoteAuthConnecting = false
	m.AddSkillActive = false
	m.AddSkillError = ""
	m.ConfigLLMActive = false
	m.ConfigLLMChecking = false
	m.ConfigLLMError = ""
	m.RemoteAuthStep = ""
	m.RemoteAuthTarget = ""
	m.RemoteAuthError = ""
	m.RemoteAuthUsername = ""
	m.UpdateSkillActive = false
	m.UpdateSkillError = ""
	if refocusInput {
		// Esc path keeps prior behavior: always refocus main input after closing overlays.
		m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleOverlayShowMsg(msg OverlayShowMsg) (Model, tea.Cmd) {
	m.OverlayActive = true
	m.OverlayTitle = msg.Title
	m.OverlayContent = msg.Content
	m.OverlayViewport = viewport.New(m.Width-4, min(m.Height-6, 20))
	m.OverlayViewport.SetContent(m.OverlayContent)
	return m, nil
}

func (m Model) handleOverlayCloseMsg() (Model, tea.Cmd) {
	return m.closeOverlayCommon(false)
}

// handleOverlayKey routes key input when overlay is active.
func (m Model) handleOverlayKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.currentUIState() != uiStateOverlay {
		return m, nil, false
	}

	for _, p := range overlayKeyProviders {
		if m2, cmd, handled := p(m, key, msg); handled {
			return m2, cmd, true
		}
	}

	switch key {
	case "esc":
		m, cmd := m.closeOverlayCommon(true)
		return m, cmd, true
	default:
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
		// Generic overlay: pass up/down/pgup/pgdown.
		var cmd tea.Cmd
		m.OverlayViewport, cmd = m.OverlayViewport.Update(msg)
		return m, cmd, true
	}
}
