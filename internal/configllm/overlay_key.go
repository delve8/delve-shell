package configllm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const configLLMFieldCount = 5

func handleOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	if !m.ConfigLLMActive {
		return m, nil, false
	}

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
			m.ConfigLLMError = i18n.T("en", i18n.KeyConfigLLMModelRequired)
			return m, nil, true
		}
		m = m.ApplyConfigLLMFromOverlayStart(baseURL, apiKey, model, maxMessagesStr, maxCharsStr)
		if !m.ConfigLLMChecking {
			return m, nil, true
		}
		return m, ui.RunConfigLLMCheckCmd(), true
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
