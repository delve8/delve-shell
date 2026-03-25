package inputlifecycletype

// SubmissionRouter routes a normalized submission into a processor and returns a result.
type SubmissionRouter interface {
	Route(InputSubmission) (ProcessResult, error)
}

// SubmissionProcessor handles one class of submissions within the unified lifecycle.
type SubmissionProcessor interface {
	CanProcess(InputSubmission) bool
	Process(InputSubmission) (ProcessResult, error)
}

// OutputAdapter converts a process result into concrete UI/system-facing effects.
type OutputAdapter interface {
	Apply(ProcessResult) error
}

// PreInputMode describes the current pre-submit interaction mode.
type PreInputMode string

const (
	PreInputModePlain PreInputMode = "plain"
	PreInputModeSlash PreInputMode = "slash"
)

// PreInputState summarizes the input-phase state used to build a submission on Enter.
type PreInputState struct {
	Mode          PreInputMode
	InputValue    string
	SelectedIndex int
	HasCandidates bool
}

// PreInputEngine updates pre-submit state and forms submissions on Enter.
type PreInputEngine interface {
	OnInputChanged(current string) PreInputState
	OnEnter(current string, selectedIndex int) (InputSubmission, bool)
}
