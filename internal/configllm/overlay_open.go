package configllm

import (
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/configsvc"
	"delve-shell/internal/ui"
)

// openOverlay opens Config LLM dialog with current config values pre-filled.
func openOverlay(m ui.Model) ui.Model {
	cfg := configsvc.LoadOrDefault()
	var st overlayState
	st.Active = true
	st.Checking = false
	st.Error = ""
	st.FieldIndex = 0
	st.BaseURLInput = textinput.New()
	st.BaseURLInput.Placeholder = "https://api.openai.com/v1 (optional)"
	st.BaseURLInput.SetValue(cfg.LLM.BaseURL)
	st.BaseURLInput.Focus()
	st.ApiKeyInput = textinput.New()
	st.ApiKeyInput.Placeholder = "sk-... or $API_KEY"
	st.ApiKeyInput.EchoMode = textinput.EchoPassword
	st.ApiKeyInput.SetValue(cfg.LLM.APIKey)
	st.ApiKeyInput.Blur()
	st.ModelInput = textinput.New()
	st.ModelInput.Placeholder = "gpt-4o-mini (optional)"
	st.ModelInput.SetValue(cfg.LLM.Model)
	st.ModelInput.Blur()
	st.MaxMessagesInput = textinput.New()
	st.MaxMessagesInput.Placeholder = ""
	if cfg.LLM.MaxContextMessages > 0 {
		st.MaxMessagesInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextMessages))
	}
	st.MaxMessagesInput.Blur()
	st.MaxCharsInput = textinput.New()
	st.MaxCharsInput.Placeholder = ""
	if cfg.LLM.MaxContextChars > 0 {
		st.MaxCharsInput.SetValue(strconv.Itoa(cfg.LLM.MaxContextChars))
	}
	st.MaxCharsInput.Blur()
	setOverlayState(st)
	return m.OpenOverlayFeature("config_llm", i18n.T("en", i18n.KeyConfigLLMTitle), "")
}
