package configllm

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

// buildConfigModelOverlayContent renders the Config Model modal body (inputs + hints).
func buildConfigModelOverlayContent() (string, bool) {
	st := getOverlayState()
	if !st.Active {
		return "", false
	}
	var b strings.Builder
	if st.Checking {
		b.WriteString(ui.SuggestStyleRender(i18n.T(i18n.KeyConfigModelChecking)) + "\n\n")
	} else if st.Error != "" {
		b.WriteString(ui.ErrStyleRender(st.Error) + "\n\n")
	}
	b.WriteString(i18n.T(i18n.KeyConfigModelBaseURLLabel) + "\n")
	b.WriteString(st.BaseURLInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18n.KeyConfigModelApiKeyLabel) + "\n")
	b.WriteString(st.ApiKeyInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18n.KeyConfigModelModelLabel) + "\n")
	b.WriteString(st.ModelInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18n.KeyConfigModelMaxMessagesLabel) + "\n")
	b.WriteString(st.MaxMessagesInput.View())
	b.WriteString("\n\n")
	b.WriteString(i18n.T(i18n.KeyConfigModelMaxCharsLabel) + "\n")
	b.WriteString(st.MaxCharsInput.View())
	b.WriteString("\n\n")
	b.WriteString(ui.RenderOverlayFormFooterHint())
	return b.String(), true
}
