package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	// /run prefix + slash-selected fill registered in internal/run.

	registerSlashPrefix("/config auto-run ", SlashPrefixDispatchEntry{
		Prefix: "/config auto-run ",
		Handle: func(mm Model, rest string) (Model, tea.Cmd, bool) {
			mm = mm.applyConfigAllowlistAutoRun(strings.TrimSpace(rest))
			return mm, nil, true
		},
	})

	// NOTE: skill/remote/session/configllm prefix handlers moved to feature packages.
}
