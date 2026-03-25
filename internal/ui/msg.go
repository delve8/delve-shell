package ui

import "delve-shell/internal/uivm"

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

// SlashSubmitRelayMsg carries structured slash intent from host controller back into Update (§10.8.1).
// Handlers must call executeMainEnterCommandNoRelay, not handleMainEnterCommand, to avoid relay recursion.
type SlashSubmitRelayMsg struct {
	RawLine            string
	SlashSelectedIndex int
	// InputLine is set when relaying slash-mode Enter (preserve raw input buffer).
	InputLine string
}
