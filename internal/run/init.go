// Package run registers /run slash prefix and fill-only selection for SlashRunUsageOption.
// internal/ui tests cannot import this package (run → ui cycle); they mirror the selected handler there.
package run

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui"
)

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

func delveMsg(lang, msg string) string {
	return i18n.T(lang, i18n.KeyDelveLabel) + " " + msg
}

func init() {
	ui.RegisterSlashPrefix("/run ", ui.SlashPrefixDispatchEntry{
		Prefix: "/run ",
		Handle: func(mm ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			cmd := strings.TrimSpace(rest)
			if cmd != "" {
				mm.Host.PublishExecDirect(cmd)
			} else {
				lang := "en"
				mm.Messages = append(mm.Messages, errStyle.Render(delveMsg(lang, i18n.T(lang, i18n.KeyUsageRun))))
			}
			return mm, nil, true
		},
	})

	ui.RegisterSlashSelectedProvider(func(m ui.Model, chosen string) (ui.Model, tea.Cmd, bool) {
		if chosen != ui.SlashRunUsageOption {
			return m, nil, false
		}
		m.Input.SetValue("/run ")
		m.Input.CursorEnd()
		return m, nil, true
	})
}
