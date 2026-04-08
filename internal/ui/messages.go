package ui

import "delve-shell/internal/ui/uivm"

// ChoiceCardShowMsg asks the UI to show a pending choice card (approval or sensitive confirmation).
// Exactly one of PendingApproval/PendingSensitive should be non-nil.
type ChoiceCardShowMsg struct {
	PendingApproval  *uivm.PendingApproval
	PendingSensitive *uivm.PendingSensitive
}

// OfflinePasteShowMsg asks the UI to show the offline paste-back dialog for a proposed command.
type OfflinePasteShowMsg struct {
	Pending *uivm.PendingOfflinePaste
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

// OverlayShowMsg opens the generic text overlay (same chrome as /help): scroll with PgUp/PgDn, Esc closes.
type OverlayShowMsg struct {
	Title   string
	Content string
}

// HistoryPreviewOverlayMsg opens the /history preview modal; Enter sends SessionSwitch for SessionID, Esc closes without switching.
// When Lines is non-empty, the UI builds styled body text from Lines (aligned with the main transcript, full commands/output).
// Content is used when Lines is empty (tests and legacy callers).
type HistoryPreviewOverlayMsg struct {
	SessionID string
	Title     string
	Content   string
	Lines     []uivm.Line
}

// transcriptPrintedMsg is emitted after a batch of tea.Println lines has been applied to the
// scrollback region, so printedMessages stays in sync with the terminal (avoids layout drift).
type transcriptPrintedMsg struct {
	upTo int
}

// offlinePasteCopyAckClearMsg clears OfflinePasteState.copyFeedback after a short delay.
type offlinePasteCopyAckClearMsg struct{}

// CommandExecutionStateMsg toggles [EXECUTING] chrome and input lock while a command runs.
type CommandExecutionStateMsg struct {
	Active bool
}

// ExecStreamWindowOpenMsg marks the start of streamed command output (after Run: is appended).
// The UI reserves a small preview band until [ExecStreamFlushMsg].
type ExecStreamWindowOpenMsg struct{}

// ExecStreamPreviewMsg carries one stdout/stderr line for the live preview (not transcript yet).
type ExecStreamPreviewMsg struct {
	Line   string
	Stderr bool
}

// ExecStreamFlushMsg appends buffered stream output as one transcript block (joined with newlines), truncating
// to the last preview-sized chunk of lines with a hint when long; when Sensitive, runs history.RedactText on
// body and tail before display. Full stdout/stderr is still stored in session history when the tool stores results.
type ExecStreamFlushMsg struct {
	Sensitive bool
	Tail      string
}
