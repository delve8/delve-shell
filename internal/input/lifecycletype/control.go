package inputlifecycletype

// ControlAction is the resolved runtime control behavior.
type ControlAction string

const (
	ControlCancelProcessing       ControlAction = "cancel_processing"
	ControlCancelCommandExecution ControlAction = "cancel_command_execution"
	ControlCloseOverlay           ControlAction = "close_overlay"
	ControlClearPreInput          ControlAction = "clear_pre_input"
	ControlQuit                   ControlAction = "quit"
	ControlInterrupt              ControlAction = "interrupt"
)

// ControlSignal identifies the incoming control intent before state-aware resolution.
type ControlSignal string

const (
	ControlSignalEsc       ControlSignal = "esc"
	ControlSignalQuit      ControlSignal = "quit"
	ControlSignalInterrupt ControlSignal = "interrupt"
)

// ControlContext captures the state needed to resolve Esc-like control priority.
type ControlContext struct {
	HasActiveOverlay bool
	HasPreInputState bool
	CommandExecuting bool
	WaitingForAI     bool
}

// ResolveEscAction applies the current Esc priority rule:
// overlay -> pre-input state -> cancel in-flight command -> cancel LLM processing -> no-op.
func ResolveEscAction(ctx ControlContext) (ControlAction, bool) {
	switch {
	case ctx.HasActiveOverlay:
		return ControlCloseOverlay, true
	case ctx.HasPreInputState:
		return ControlClearPreInput, true
	case ctx.CommandExecuting:
		return ControlCancelCommandExecution, true
	case ctx.WaitingForAI:
		return ControlCancelProcessing, true
	default:
		return "", false
	}
}
