package configllm

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

const configLLMFieldCount = 5

func handleOverlayKey(m ui.Model, key string, msg tea.KeyMsg) (ui.Model, tea.Cmd, bool) {
	if !m.ConfigLLM.Active {
		return m, nil, false
	}

	switch key {
	case "up", "down":
		dir := 1
		if key == "up" {
			dir = -1
		}
		m.ConfigLLM.FieldIndex = (m.ConfigLLM.FieldIndex + dir + configLLMFieldCount) % configLLMFieldCount
		m.ConfigLLM.BaseURLInput.Blur()
		m.ConfigLLM.ApiKeyInput.Blur()
		m.ConfigLLM.ModelInput.Blur()
		m.ConfigLLM.MaxMessagesInput.Blur()
		m.ConfigLLM.MaxCharsInput.Blur()
		switch m.ConfigLLM.FieldIndex {
		case 0:
			m.ConfigLLM.BaseURLInput.Focus()
		case 1:
			m.ConfigLLM.ApiKeyInput.Focus()
		case 2:
			m.ConfigLLM.ModelInput.Focus()
		case 3:
			m.ConfigLLM.MaxMessagesInput.Focus()
		case 4:
			m.ConfigLLM.MaxCharsInput.Focus()
		}
		return m, nil, true
	case "enter":
		if m.ConfigLLM.Checking {
			return m, nil, true
		}
		baseURL := strings.TrimSpace(m.ConfigLLM.BaseURLInput.Value())
		apiKey := strings.TrimSpace(m.ConfigLLM.ApiKeyInput.Value())
		model := strings.TrimSpace(m.ConfigLLM.ModelInput.Value())
		maxMessagesStr := strings.TrimSpace(m.ConfigLLM.MaxMessagesInput.Value())
		maxCharsStr := strings.TrimSpace(m.ConfigLLM.MaxCharsInput.Value())
		if model == "" {
			m.ConfigLLM.Error = i18n.T("en", i18n.KeyConfigLLMModelRequired)
			return m, nil, true
		}
		m = applyConfigLLMFromOverlayStart(m, baseURL, apiKey, model, maxMessagesStr, maxCharsStr)
		if !m.ConfigLLM.Checking {
			return m, nil, true
		}
		return m, runConfigLLMCheckCmd(), true
	}

	var cmd tea.Cmd
	switch m.ConfigLLM.FieldIndex {
	case 0:
		m.ConfigLLM.BaseURLInput, cmd = m.ConfigLLM.BaseURLInput.Update(msg)
	case 1:
		m.ConfigLLM.ApiKeyInput, cmd = m.ConfigLLM.ApiKeyInput.Update(msg)
	case 2:
		m.ConfigLLM.ModelInput, cmd = m.ConfigLLM.ModelInput.Update(msg)
	case 3:
		m.ConfigLLM.MaxMessagesInput, cmd = m.ConfigLLM.MaxMessagesInput.Update(msg)
	case 4:
		m.ConfigLLM.MaxCharsInput, cmd = m.ConfigLLM.MaxCharsInput.Update(msg)
	}
	return m, cmd, true
}
