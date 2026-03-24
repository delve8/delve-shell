package run

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

func init() {
	ui.RegisterSlashPrefix("/config auto-run ", ui.SlashPrefixDispatchEntry{
		Prefix: "/config auto-run ",
		Handle: func(m ui.Model, rest string) (ui.Model, tea.Cmd, bool) {
			m = applyConfigAllowlistAutoRun(m, strings.TrimSpace(rest))
			return m, nil, true
		},
	})
}
