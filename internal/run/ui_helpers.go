package run

import (
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/i18n"
)

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

func delveMsg(lang, msg string) string {
	return i18n.T(lang, i18n.KeyDelveLabel) + " " + msg
}
