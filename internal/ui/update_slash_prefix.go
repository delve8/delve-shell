package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

// dispatchSlashPrefix handles slash commands with arguments.
// It is intended for the Enter-submit path where input is already consumed.
func (m Model) dispatchSlashPrefix(text string) (Model, tea.Cmd, bool) {
	for _, e := range slashPrefixDispatchRegistry {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			return e.Handle(m, rest)
		}
	}
	return m, nil, false
}

func init() {
	// NOTE: order matters for prefix overlaps. Keep it explicit and deterministic.
	registerSlashPrefix("/run ", SlashPrefixDispatchEntry{
		Prefix: "/run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			cmd := strings.TrimSpace(rest)
			if mm.ExecDirectChan != nil && cmd != "" {
				mm.ExecDirectChan <- cmd
			} else if cmd == "" {
				mm.Messages = append(mm.Messages, errStyle.Render(mm.delveMsg(i18n.T(mm.getLang(), i18n.KeyUsageRun))))
			}
			return mm, nil, true
		},
	})

	registerSlashPrefix("/config auto-run ", SlashPrefixDispatchEntry{
		Prefix: "/config auto-run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigAllowlistAutoRun(strings.TrimSpace(rest))
			return mm, nil, true
		},
	})

	// NOTE: skill/remote/session/configllm prefix handlers moved to feature packages.
}
