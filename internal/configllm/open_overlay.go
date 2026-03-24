package configllm

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/configsvc"
	"delve-shell/internal/ui"
)

// openOverlay opens Config LLM dialog with current config values pre-filled.
func openOverlay(m ui.Model) ui.Model {
	cfg := configsvc.LoadOrDefault()
	m.OverlayActive = true
	m.OverlayTitle = i18n.T("en", i18n.KeyConfigLLMTitle)
	m.ConfigLLMActive = true
	m.ConfigLLMChecking = false
	m.ConfigLLMError = ""
	m.ConfigLLMFieldIndex = 0
	m.ConfigLLMBaseURLInput = textinput.New()
	m.ConfigLLMBaseURLInput.Placeholder = "https://api.openai.com/v1 (optional)"
	m.ConfigLLMBaseURLInput.SetValue(cfg.LLM.BaseURL)
	m.ConfigLLMBaseURLInput.Focus()
	m.ConfigLLMApiKeyInput = textinput.New()
	m.ConfigLLMApiKeyInput.Placeholder = "sk-... or $API_KEY"
	m.ConfigLLMApiKeyInput.EchoMode = textinput.EchoPassword
	m.ConfigLLMApiKeyInput.SetValue(cfg.LLM.APIKey)
	m.ConfigLLMApiKeyInput.Blur()
	m.ConfigLLMModelInput = textinput.New()
	m.ConfigLLMModelInput.Placeholder = "gpt-4o-mini (optional)"
	m.ConfigLLMModelInput.SetValue(cfg.LLM.Model)
	m.ConfigLLMModelInput.Blur()
	m.ConfigLLMMaxMessagesInput = textinput.New()
	m.ConfigLLMMaxMessagesInput.Placeholder = ""
	if cfg.LLM.MaxContextMessages > 0 {
		m.ConfigLLMMaxMessagesInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextMessages))
	}
	m.ConfigLLMMaxMessagesInput.Blur()
	m.ConfigLLMMaxCharsInput = textinput.New()
	m.ConfigLLMMaxCharsInput.Placeholder = ""
	if cfg.LLM.MaxContextChars > 0 {
		m.ConfigLLMMaxCharsInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextChars))
	}
	m.ConfigLLMMaxCharsInput.Blur()
	return m
}

func registerSlashExact() {
	ui.RegisterSlashExact("/config llm", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return openOverlay(m), nil
		},
		ClearInput: true,
	})
}
