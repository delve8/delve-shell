package configllm

// CheckDoneMsg is sent when the async model config check (after save) finishes.
// ErrText non-empty means check failed; CorrectedBaseURL non-empty means /v1 was added and config was updated.
type CheckDoneMsg struct {
	ErrText          string
	CorrectedBaseURL string
}
