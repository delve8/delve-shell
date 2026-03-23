package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleMainScrollKey(key string, msg tea.KeyMsg, inputVal string) (Model, tea.Cmd, bool) {
	if key != "up" && key != "down" && key != "pgup" && key != "pgdown" {
		return m, nil, false
	}
	inSlash := strings.HasPrefix(inputVal, "/")
	// scroll keys: Up/Down change selection in slash mode, else go to viewport with PgUp/PgDown
	if inSlash && (key == "up" || key == "down") {
		opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
		vis := visibleSlashOptions(inputVal, opts)
		if len(vis) > 0 {
			if m.SlashSuggestIndex >= len(vis) {
				m.SlashSuggestIndex = 0
			}
			if key == "down" {
				m.SlashSuggestIndex = (m.SlashSuggestIndex + 1) % len(vis)
			} else {
				m.SlashSuggestIndex = (m.SlashSuggestIndex - 1 + len(vis)) % len(vis)
			}
		}
		return m, nil, true
	}
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd, true
}

func (m Model) captureSlashSelectionForEnter(inputVal string, text string) (Model, string, int, bool) {
	slashSelectedPath := ""
	slashSelectedIndex := -1
	if !strings.HasPrefix(inputVal, "/") {
		return m, slashSelectedPath, slashSelectedIndex, false
	}
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.CurrentSessionPath, m.LocalRunCommands, m.RemoteRunCommands, m.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) > 0 && m.SlashSuggestIndex < len(vis) {
		chosen := opts[vis[m.SlashSuggestIndex]].Cmd
		// chosen != text => fill selection only, do not execute or add to View
		if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
			m.Input.SetValue(slashChosenToInputValue(chosen))
			m.Input.CursorEnd()
			m.SlashSuggestIndex = 0 // reset so next Enter (new opts set, e.g. skill list) uses index 0
			return m, "", -1, true
		}
		slashSelectedIndex = m.SlashSuggestIndex
		if opts[vis[m.SlashSuggestIndex]].Path != "" {
			slashSelectedPath = opts[vis[m.SlashSuggestIndex]].Path
		}
	}
	return m, slashSelectedPath, slashSelectedIndex, false
}
