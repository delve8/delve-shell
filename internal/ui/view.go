package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

// View implements tea.Model.
func (m Model) View() string {
	lang := m.getLang()
	sepW := m.Layout.Width
	if sepW <= 0 {
		sepW = 40
	}
	sepLine := separatorStyle.Render(strings.Repeat("─", sepW))
	header := m.titleLine() + "\n" + sepLine + "\n"

	inChoice := m.hasPendingApproval()
	if m.Layout.Height <= 4 {
		out := header + m.buildContent() + "\n" + m.Input.View()
		out += m.waitingLineBelowInput(lang)
		return out
	}
	// Base viewport height: leave room for header, separator, input line, and slash/choice dropdown (the two lines at bottom are for input + suggestions).
	vh := m.Layout.Height - 10
	if vh < 1 {
		vh = 1
	}
	m.Viewport.Width = m.Layout.Width
	m.Viewport.Height = vh
	out := header
	out += m.Viewport.View()
	out += "\n" + sepLine + "\n"
	out += m.Input.View()
	if inChoice {
		out += m.choiceLinesBelowInput(lang)
	} else {
		out += m.slashDropdownBelowInput(lang)
	}
	out += m.waitingLineBelowInput(lang)

	// Render overlay on top if active.
	if m.Overlay.Active {
		out = m.renderOverlay(out)
	}
	return out
}

// appendSuggestedLine appends the run line and copy hint for a suggested command (when dismissing the card).
func (m *Model) appendSuggestedLine(command, lang string) {
	tag := i18n.T(lang, i18n.KeyRunTagSuggested)
	line := i18n.T(lang, i18n.KeyRunLabel) + command + " (" + tag + ")"
	w := m.contentWidth()
	m.Messages = append(m.Messages, execStyle.Render(wrapString(line, w)))
	m.Messages = append(m.Messages, hintStyle.Render(i18n.T(lang, i18n.KeySuggestedCopyHint)))
}
