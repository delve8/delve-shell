package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

func (m Model) handleNewSessionCommandIfNeeded(text string) (Model, bool) {
	// /new sends to run loop only; do not append to Messages
	if text != "/new" {
		return m, false
	}
	if m.SubmitChan != nil {
		m.SubmitChan <- text
	}
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.SlashSuggestIndex = 0
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	return m, true
}

func (m Model) appendUserInputLine(text string) Model {
	userLine := i18n.T(m.getLang(), i18n.KeyUserLabel) + text
	w := m.Width
	if w <= 0 {
		w = 80
	}
	sepW := w
	sepLine := separatorStyle.Render(strings.Repeat("─", sepW))
	if len(m.Messages) > 0 && m.Messages[len(m.Messages)-1] != sepLine {
		m.Messages = append(m.Messages, sepLine)
	}
	m.Messages = append(m.Messages, wrapString(userLine, w))
	m.Messages = append(m.Messages, "") // blank line before command or AI reply
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.SlashSuggestIndex = 0
	return m
}
