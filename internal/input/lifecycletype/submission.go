package inputlifecycletype

// SubmissionKind classifies the top-level input path after Enter is submitted.
type SubmissionKind string

const (
	SubmissionChat    SubmissionKind = "chat"
	SubmissionSlash   SubmissionKind = "slash"
	SubmissionControl SubmissionKind = "control"
)

// SubmissionSource describes where a submission originated.
type SubmissionSource string

const (
	SourceMainEnter       SubmissionSource = "main_enter"
	SourceSlashEarlyEnter SubmissionSource = "slash_early_enter"
	SourceKeyboardSignal  SubmissionSource = "keyboard_signal"
	SourceProgrammatic    SubmissionSource = "programmatic"
)

// InputSubmission is the structured input object emitted after Enter or an equivalent control submit.
type InputSubmission struct {
	Kind   SubmissionKind
	Source SubmissionSource

	// RawText is the normalized submitted text.
	RawText string
	// SessionDisplayText, when non-empty, is what should be recorded in session history and shown as the user line
	// instead of RawText (e.g. /skill … while RawText carries the LLM payload).
	SessionDisplayText string
	// SkillInvocationSkillName is the /skill <name> directory name when this chat turn was started by /skill; empty otherwise.
	// Used so run_skill for that skill can skip a second approval in the same LLM turn.
	SkillInvocationSkillName string
	// InputLine preserves the raw input buffer for pre-input flows such as slash early Enter.
	InputLine string
	// SelectedIndex is meaningful for slash submissions and should be -1 when not applicable.
	SelectedIndex int
	// SelectedCmd preserves the exact slash row selected at submit time when relevant.
	SelectedCmd string
	// SelectedFill preserves the selected row's fill value when it differs from the visible command.
	SelectedFill string
	// ControlSignal is meaningful for control submissions.
	ControlSignal ControlSignal
}
