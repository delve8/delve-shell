package inputlifecycletype

// OutputEventKind identifies the UI-facing effect emitted by processing.
type OutputEventKind string

const (
	OutputTranscriptAppend OutputEventKind = "transcript_append"
	OutputOverlayOpen      OutputEventKind = "overlay_open"
	OutputOverlayClose     OutputEventKind = "overlay_close"
	OutputPreInputSet      OutputEventKind = "pre_input_set"
	OutputPreInputClear    OutputEventKind = "pre_input_clear"
	OutputStatusChange     OutputEventKind = "status_change"
	OutputCommandExecution OutputEventKind = "command_execution"
	OutputApprovalOpen     OutputEventKind = "approval_open"
	OutputErrorNotice      OutputEventKind = "error_notice"
	OutputQuit             OutputEventKind = "quit"
)

// TranscriptPayload appends semantic transcript lines.
type TranscriptPayload struct {
	Lines []TranscriptLine
}

// TranscriptLineKind is the lifecycle-level semantic kind of a transcript line.
type TranscriptLineKind int

const (
	TranscriptLinePlain TranscriptLineKind = iota
	TranscriptLineBlank
	TranscriptLineSeparator
	TranscriptLineUser
	TranscriptLineAI
	TranscriptLineSystemSuggest
	TranscriptLineSystemError
	TranscriptLineExec
	TranscriptLineResult
)

// TranscriptLine is one semantic transcript line.
type TranscriptLine struct {
	Kind TranscriptLineKind
	Text string
}

// PreInputPayload describes a requested pre-input mutation, such as filling a slash command.
type PreInputPayload struct {
	Value string
}

// OverlayPayload describes an overlay open/close effect.
type OverlayPayload struct {
	Key     string
	Title   string
	Content string
	Params  map[string]string
	// Markdown when true: Content is GitHub-flavored Markdown rendered for the help-style scroll overlay.
	Markdown bool
}

// StatusPayload describes a UI status update such as idle/running/pending.
type StatusPayload struct {
	Key string
}

// CommandExecutionPayload toggles command-run UI lock ([EXECUTING]) without changing LLM waiting state.
type CommandExecutionPayload struct {
	Active bool
}

// ApprovalPayload describes a request to show an approval/sensitive confirmation surface.
type ApprovalPayload struct {
	Title       string
	Command     string
	Description string
}

// ErrorPayload describes a user-visible error notice.
type ErrorPayload struct {
	Message string
}

// OutputEvent is the unified result emitted from processing before UI adaptation.
type OutputEvent struct {
	Kind OutputEventKind

	Text string

	Transcript  *TranscriptPayload
	PreInput    *PreInputPayload
	Overlay     *OverlayPayload
	Status      *StatusPayload
	CommandExec *CommandExecutionPayload
	Approval    *ApprovalPayload
	Error       *ErrorPayload
}
