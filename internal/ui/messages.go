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
	// ClearWaitingForAI clears the post-submit LLM / "processing" state (title bar + footer).
	ClearWaitingForAI bool
}

// TranscriptReplaceMsg replaces the whole transcript with semantic lines.
type TranscriptReplaceMsg struct {
	Lines []uivm.Line
}

// transcriptPrintedMsg is emitted after a batch of tea.Println lines has been applied to the
// scrollback region, so printedMessages stays in sync with the terminal (avoids layout drift).
type transcriptPrintedMsg struct {
	upTo int
}
