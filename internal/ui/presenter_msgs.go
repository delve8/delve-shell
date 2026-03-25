package ui

import tea "github.com/charmbracelet/bubbletea"

// Presenter message factories (used by uipresenter; keeps struct literals out of the host→TUI boundary).

func NewOverlayCloseMsg() tea.Msg { return OverlayCloseMsg{} }

func NewOverlayShowMsg(title, content string) tea.Msg {
	return OverlayShowMsg{Title: title, Content: content}
}
