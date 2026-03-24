package ui

import (
	"strings"

	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	trimmed := strings.TrimSpace(inputVal)
	if trimmed == "" {
		return m, nil, false
	}
	if m2, cmd, handled := m.dispatchSlashExact(trimmed); handled {
		return m2, cmd, true
	}
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	selected, ok := slashview.SelectedByVisibleIndex(toSlashViewOptions(opts), vis, m.Interaction.SlashSuggestIndex)
	result := slashflow.EvaluateSlashEnter(inputVal, trimmed, selected, ok)
	switch result.Action {
	case slashflow.EnterKeyDispatchExactChosen:
		if m2, cmd, handled := m.dispatchSlashExact(selected.Cmd); handled {
			return m2, cmd, true
		}
	case slashflow.EnterKeyFillOnly:
		m.Input.SetValue(result.Fill)
		m.Input.CursorEnd()
		m.Interaction.SlashSuggestIndex = 0
		return m, nil, true
	}
	return m, nil, false
}
