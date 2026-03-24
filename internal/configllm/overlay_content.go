package configllm

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// buildConfigLLMOverlayContent renders the Config LLM modal body (inputs + hints).
func buildConfigLLMOverlayContent(m ui.Model) (string, bool) {
	if !m.ConfigLLMActive {
		return "", false
	}
	lang := "en"
	var b strings.Builder
	if m.ConfigLLMChecking {
		b.WriteString(ui.SuggestStyleRender(i18n.T(lang, i18n.KeyConfigLLMChecking)) + "\n\n")
	} else if m.ConfigLLMError != "" {
		b.WriteString(ui.ErrStyleRender(m.ConfigLLMError) + "\n\n")
	}
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMBaseURLLabel) + "\n")
	b.WriteString(m.ConfigLLMBaseURLInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMApiKeyLabel) + "\n")
	b.WriteString(m.ConfigLLMApiKeyInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMModelLabel) + "\n")
	b.WriteString(m.ConfigLLMModelInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxMessagesLabel) + "\n")
	b.WriteString(m.ConfigLLMMaxMessagesInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxCharsLabel) + "\n")
	b.WriteString(m.ConfigLLMMaxCharsInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMHint))
	return b.String(), true
}
