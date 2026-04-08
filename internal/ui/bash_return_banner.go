package ui

import (
	"delve-shell/internal/i18n"
)

// BashReturnTranscriptLine is a styled transcript row to append after /bash subshell exits,
// so scrollback shows a clear boundary before the restored session continues.
func BashReturnTranscriptLine() string {
	i18n.SetLang(languageFromConfig())
	msg := i18n.T(i18n.KeyInfoLabel) + i18n.T(i18n.KeyBashReturnNotice)
	return infoStyle.Render(msg)
}
