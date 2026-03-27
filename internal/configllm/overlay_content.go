package configllm

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// buildConfigLLMOverlayContent renders the Config LLM modal body (inputs + hints).
func buildConfigLLMOverlayContent() (string, bool) {
	st := getOverlayState()
	if !st.Active {
		return "", false
	}
	lang := "en"
	var b strings.Builder
	if st.Checking {
		b.WriteString(ui.SuggestStyleRender(i18n.T(lang, i18n.KeyConfigLLMChecking)) + "\n\n")
	} else if st.Error != "" {
		b.WriteString(ui.ErrStyleRender(st.Error) + "\n\n")
	}
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMBaseURLLabel) + "\n")
	b.WriteString(st.BaseURLInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMApiKeyLabel) + "\n")
	b.WriteString(st.ApiKeyInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMModelLabel) + "\n")
	b.WriteString(st.ModelInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxMessagesLabel) + "\n")
	b.WriteString(st.MaxMessagesInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(lang, i18n.KeyConfigLLMMaxCharsLabel) + "\n")
	b.WriteString(st.MaxCharsInput.View())
	b.WriteString("\n\n")
	b.WriteString(ui.RenderOverlayFormFooterHint(lang))
	return b.String(), true
}
