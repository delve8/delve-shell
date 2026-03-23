package hostfsm

// Host coordination states (orthogonal to Bubble Tea Model).
const (
	StateIdle       State = "idle"
	StateLLMRunning State = "llm_running"
)

const (
	EvtLLMRunStart Event = "llm_run_start"
	EvtLLMRunEnd   Event = "llm_run_end"
)
