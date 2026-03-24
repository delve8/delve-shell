package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) handleMainScrollKey(key string, msg tea.KeyMsg, inputVal string) (Model, tea.Cmd, bool) {
	if key != "up" && key != "down" && key != "pgup" && key != "pgdown" {
		return m, nil, false
	}
	inSlash := strings.HasPrefix(inputVal, "/")
	// scroll keys: Up/Down change selection in slash mode, else go to viewport with PgUp/PgDown
	if inSlash && (key == "up" || key == "down") {
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

func (m Model) captureSlashSelectionForEnter(inputVal string, text string) (Model, string, int, bool) {
	slashSelectedPath := ""
	slashSelectedIndex := -1
	if !strings.HasPrefix(inputVal, "/") {
		return m, slashSelectedPath, slashSelectedIndex, false
	}
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	if len(vis) > 0 && m.Interaction.SlashSuggestIndex < len(vis) {
		chosen := opts[vis[m.Interaction.SlashSuggestIndex]].Cmd
		// chosen != text => fill selection only, do not execute or add to View
		if (chosen == text || strings.HasPrefix(chosen, text)) && chosen != text {
			m.Input.SetValue(slashChosenToInputValue(chosen))
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0 // reset so next Enter (new opts set, e.g. skill list) uses index 0
			return m, "", -1, true
		}
		slashSelectedIndex = m.Interaction.SlashSuggestIndex
		if opts[vis[m.Interaction.SlashSuggestIndex]].Path != "" {
			slashSelectedPath = opts[vis[m.Interaction.SlashSuggestIndex]].Path
		}
	}
	return m, slashSelectedPath, slashSelectedIndex, false
}

func (m Model) handleMainInputUpdate(msg tea.KeyMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	m.syncSlashSuggestIndex()
	return m, cmd
}

// syncSlashSuggestIndex keeps slash suggestion selection in a valid range
// when the user edits slash input.
func (m *Model) syncSlashSuggestIndex() {
	if !strings.HasPrefix(m.Input.Value(), "/") {
		return
	}
	inputVal := m.Input.Value()
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	// Session list (Path set): do not reset index on every keystroke so user can pick another session with Enter
	if len(opts) == 0 || opts[0].Path == "" {
		m.Interaction.SlashSuggestIndex = 0
	}
	if len(vis) > 0 && m.Interaction.SlashSuggestIndex >= len(vis) {
		m.Interaction.SlashSuggestIndex = 0
	}
}

func (m Model) handleNewSessionCommandIfNeeded(text string) (Model, bool) {
	// /new sends to run loop only; do not append to Messages
	if text != "/new" {
		return m, false
	}
	if m.Ports.SubmitChan != nil {
		m.Ports.SubmitChan <- text
	}
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.SlashSuggestIndex = 0
	m = m.RefreshViewport()
	return m, true
}

func (m Model) appendUserInputLine(text string) Model {
	userLine := i18n.T(m.getLang(), i18n.KeyUserLabel) + text
	w := m.contentWidth()
	sepW := w
	sepLine := separatorStyle.Render(strings.Repeat("─", sepW))
	if len(m.Messages) > 0 && m.Messages[len(m.Messages)-1] != sepLine {
		m.Messages = append(m.Messages, sepLine)
	}
	m.Messages = append(m.Messages, wrapString(userLine, w))
	m.Messages = append(m.Messages, "") // blank line before command or AI reply
	m = m.RefreshViewport()
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.SlashSuggestIndex = 0
	return m
}
