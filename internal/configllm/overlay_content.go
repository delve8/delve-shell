package configllm

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// buildConfigLLMOverlayContent renders the Config LLM modal body (inputs + hints).
func buildConfigLLMOverlayContent(m ui.Model) (string, bool) {
	if !m.ConfigLLM.Active {
		return "", false
	}
	lang := "en"
	var b strings.Builder
	if m.ConfigLLM.Checking {
		b.WriteString(ui.SuggestStyleRender(i18n.T(lang, i18n.KeyConfigLLMChecking)) + "\n\n")
	} else if m.ConfigLLM.Error != "" {
		b.WriteString(ui.ErrStyleRender(m.ConfigLLM.Error) + "\n\n")
	}
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMBaseURLLabel) + "\n")
	b.WriteString(m.ConfigLLM.BaseURLInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMApiKeyLabel) + "\n")
	b.WriteString(m.ConfigLLM.ApiKeyInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMModelLabel) + "\n")
	b.WriteString(m.ConfigLLM.ModelInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxMessagesLabel) + "\n")
	b.WriteString(m.ConfigLLM.MaxMessagesInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxCharsLabel) + "\n")
	b.WriteString(m.ConfigLLM.MaxCharsInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMHint))
	return b.String(), true
}
