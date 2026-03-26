package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/uivm"
)

// ChoiceCardShowMsg asks the UI to show a pending choice card (approval or sensitive confirmation).
// Exactly one of PendingApproval/PendingSensitive should be non-nil.
type ChoiceCardShowMsg struct {
	PendingApproval  *uivm.PendingApproval
	PendingSensitive *uivm.PendingSensitive
}

// TranscriptAppendMsg appends semantic transcript lines.
type TranscriptAppendMsg struct {
	Lines []uivm.Line
}

// TranscriptReplaceMsg replaces the whole transcript with semantic lines.
type TranscriptReplaceMsg struct {
	Lines []uivm.Line
}

// OverlayCloseMsg closes any active overlay.
type OverlayCloseMsg struct{}

// OverlayShowMsg shows an overlay with the given title and content.
type OverlayShowMsg struct {
	Title   string
	Content string
}

// OverlayOpenIntentMsg asks the UI to open a registered overlay feature by key.
type OverlayOpenIntentMsg struct {
	Key     string
	Params  map[string]string
	Title   string
	Content string
}

// LifecycleSlashExecuteMsg asks the UI to execute a slash submission locally.
type LifecycleSlashExecuteMsg struct {
	RawText       string
	InputLine     string
	SelectedIndex int
}

// Presenter message factories (used by uipresenter; keeps struct literals out of the host→TUI boundary).
func NewOverlayCloseMsg() tea.Msg { return OverlayCloseMsg{} }

func NewOverlayShowMsg(title, content string) tea.Msg {
	return OverlayShowMsg{Title: title, Content: content}
}
