package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSlashNavigationKey(key string, msg tea.KeyMsg, inputVal string) (Model, tea.Cmd, bool) {
	if key != "up" && key != "down" && key != "pgup" && key != "pgdown" {
		return m, nil, false
	}
	if key == "up" || key == "down" {
		opts := getSlashOptionsForInput(inputVal, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
		vis := visibleSlashOptions(inputVal, opts)
		if len(vis) > 0 {
			if m.Interaction.SlashSuggestIndex >= len(vis) {
				m.Interaction.SlashSuggestIndex = 0
			}
			if key == "down" {
				m.Interaction.SlashSuggestIndex = (m.Interaction.SlashSuggestIndex + 1) % len(vis)
			} else {
				m.Interaction.SlashSuggestIndex = (m.Interaction.SlashSuggestIndex - 1 + len(vis)) % len(vis)
			}
		}
		return m, nil, true
	}
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd, true
}

func (m Model) handleSlashEnterKey(inputVal string) (Model, tea.Cmd, bool) {
	if strings.TrimSpace(inputVal) == "" {
		return m, nil, false
	}
	trimmed := strings.TrimSpace(inputVal)
	// Execute exact slash commands through a single dispatch path.
	if m2, cmd, handled := m.dispatchSlashExact(trimmed); handled {
		return m2, cmd, true
	}

	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) == 0 || m.Interaction.SlashSuggestIndex >= len(vis) {
		return m, nil, false
	}

	selectedOpt := opts[vis[m.Interaction.SlashSuggestIndex]]
	chosen := selectedOpt.Cmd
	text := strings.TrimSpace(inputVal)
	if chosen == trimmed {
		if m2, cmd, handled := m.dispatchSlashExact(chosen); handled {
			return m2, cmd, true
		}
	}
	// Fill only (do not execute) when chosen extends current input.
	if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
		m.Input.SetValue(slashChosenToInputValue(chosen))
		m.Input.CursorEnd()
		m.Interaction.SlashSuggestIndex = 0
		return m, nil, true
	}
	// Otherwise, let the later Enter handler deal with execute-on-select semantics.
	return m, nil, false
}
