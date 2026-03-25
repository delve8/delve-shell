package inputlifecycletype

// OutputEventKind identifies the UI-facing effect emitted by processing.
type OutputEventKind string

const (
	OutputTranscriptAppend OutputEventKind = "transcript_append"
	OutputOverlayOpen      OutputEventKind = "overlay_open"
	OutputOverlayClose     OutputEventKind = "overlay_close"
	OutputStatusChange     OutputEventKind = "status_change"
	OutputApprovalOpen     OutputEventKind = "approval_open"
	OutputErrorNotice      OutputEventKind = "error_notice"
	OutputMessage          OutputEventKind = "message"
	OutputQuit             OutputEventKind = "quit"
)

// TranscriptPayload appends one line or block into the conversation transcript.
type TranscriptPayload struct {
	Text string
}

// OverlayPayload describes an overlay open/close effect.
type OverlayPayload struct {
	Title   string
	Content string
}

// StatusPayload describes a UI status update such as idle/running/pending.
type StatusPayload struct {
	Key string
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

// MessagePayload carries a deferred tea message without coupling lifecycle types to Bubble Tea.
type MessagePayload struct {
	Value any
}

// OutputEvent is the unified result emitted from processing before UI adaptation.
type OutputEvent struct {
	Kind OutputEventKind

	Text string

	Transcript *TranscriptPayload
	Overlay    *OverlayPayload
	Status     *StatusPayload
	Approval   *ApprovalPayload
	Error      *ErrorPayload
	Message    *MessagePayload
}
