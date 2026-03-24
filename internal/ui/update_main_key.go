package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
	"delve-shell/internal/slashview"
	"delve-shell/internal/textwrap"
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
		if next, changed := slashview.NextSuggestIndex(m.Interaction.SlashSuggestIndex, len(vis), key); changed {
			m.Interaction.SlashSuggestIndex = next
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
	selected, ok := m.selectedSlashOption(inputVal)
	if ok {
		chosen := selected.Cmd
		// chosen != text => fill selection only, do not execute or add to View
		if slashview.ShouldFillOnly(chosen, text) {
			m.Input.SetValue(slashChosenToInputValue(chosen))
			m.Input.CursorEnd()
			m.Interaction.SlashSuggestIndex = 0 // reset so next Enter (new opts set, e.g. skill list) uses index 0
			return m, "", -1, true
		}
		slashSelectedIndex = m.Interaction.SlashSuggestIndex
		if selected.Path != "" {
			slashSelectedPath = selected.Path
		}
	}
	return m, slashSelectedPath, slashSelectedIndex, false
}

func (m Model) selectedSlashOption(inputVal string) (slashview.Option, bool) {
	opts := getSlashOptionsForInput(inputVal, m.getLang(), m.Context.CurrentSessionPath, m.RunCompletion.LocalCommands, m.RunCompletion.RemoteCommands, m.Context.RemoteActive)
	vis := visibleSlashOptions(inputVal, opts)
	return slashview.SelectedByVisibleIndex(toSlashViewOptions(opts), vis, m.Interaction.SlashSuggestIndex)
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
	sepLine := renderSeparator(sepW)
	if len(m.Messages) > 0 && m.Messages[len(m.Messages)-1] != sepLine {
		m.Messages = append(m.Messages, sepLine)
	}
	m.Messages = append(m.Messages, textwrap.WrapString(userLine, w))
	m.Messages = append(m.Messages, "") // blank line before command or AI reply
	m = m.RefreshViewport()
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.Interaction.SlashSuggestIndex = 0
	return m
}
