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
	m = m.OpenOverlay(i18n.T("en", i18n.KeyConfigLLMTitle), "")
	m.ConfigLLM.Active = true
	m.ConfigLLM.Checking = false
	m.ConfigLLM.Error = ""
	m.ConfigLLM.FieldIndex = 0
	m.ConfigLLM.BaseURLInput = textinput.New()
	m.ConfigLLM.BaseURLInput.Placeholder = "https://api.openai.com/v1 (optional)"
	m.ConfigLLM.BaseURLInput.SetValue(cfg.LLM.BaseURL)
	m.ConfigLLM.BaseURLInput.Focus()
	m.ConfigLLM.ApiKeyInput = textinput.New()
	m.ConfigLLM.ApiKeyInput.Placeholder = "sk-... or $API_KEY"
	m.ConfigLLM.ApiKeyInput.EchoMode = textinput.EchoPassword
	m.ConfigLLM.ApiKeyInput.SetValue(cfg.LLM.APIKey)
	m.ConfigLLM.ApiKeyInput.Blur()
	m.ConfigLLM.ModelInput = textinput.New()
	m.ConfigLLM.ModelInput.Placeholder = "gpt-4o-mini (optional)"
	m.ConfigLLM.ModelInput.SetValue(cfg.LLM.Model)
	m.ConfigLLM.ModelInput.Blur()
	m.ConfigLLM.MaxMessagesInput = textinput.New()
	m.ConfigLLM.MaxMessagesInput.Placeholder = ""
	if cfg.LLM.MaxContextMessages > 0 {
		m.ConfigLLM.MaxMessagesInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextMessages))
	}
	m.ConfigLLM.MaxMessagesInput.Blur()
	m.ConfigLLM.MaxCharsInput = textinput.New()
	m.ConfigLLM.MaxCharsInput.Placeholder = ""
	if cfg.LLM.MaxContextChars > 0 {
		m.ConfigLLM.MaxCharsInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextChars))
	}
	m.ConfigLLM.MaxCharsInput.Blur()
	return m
}

func registerSlashExact() {
	ui.RegisterSlashExact("/config llm", ui.SlashExactDispatchEntry{
		Handle: func(m ui.Model) (ui.Model, tea.Cmd) {
			return openOverlay(m), nil
		},
		ClearInput: true,
	})
	ui.RegisterStartupOverlayProvider(func(m ui.Model) (ui.Model, tea.Cmd, bool) {
		return openOverlay(m), nil, true
	})
}
