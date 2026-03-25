package configllm

// CheckDoneMsg is sent when the async config LLM check (after save) finishes.
// ErrText non-empty means check failed; CorrectedBaseURL non-empty means /v1 was added and config was updated.
type CheckDoneMsg struct {
	ErrText          string
	CorrectedBaseURL string
}

// OpenOverlayMsg opens the config LLM overlay.
type OpenOverlayMsg struct{}

// ApplyFieldMsg applies a single `/config llm <field> ...` slash command.
type ApplyFieldMsg struct {
	Field string
	Value string
}
