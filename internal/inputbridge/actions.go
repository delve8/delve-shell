package inputbridge

import (
	"errors"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputprocess/chatproc"
	"delve-shell/internal/inputprocess/controlproc"
	"delve-shell/internal/uivm"
)

// ErrActionRejected reports that the legacy action sink refused the bridged action.
var ErrActionRejected = errors.New("input bridge: action rejected")

// ActionSink is the migration-time bridge target for legacy UI outbound intents.
type ActionSink interface {
	Send(action uivm.UIAction) bool
}

// ChatActionExecutor adapts chat submissions to the legacy submit UI action.
type ChatActionExecutor struct {
	Sink ActionSink
}

// ExecuteChat implements [chatproc.Executor].
func (e ChatActionExecutor) ExecuteChat(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	if e.Sink == nil {
		return inputlifecycletype.ProcessResult{}, ErrActionRejected
	}
	if !e.Sink.Send(uivm.UIAction{
		Kind: uivm.UIActionSubmit,
		Text: sub.RawText,
	}) {
		return inputlifecycletype.ProcessResult{}, ErrActionRejected
	}
	res := inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:   inputlifecycletype.OutputStatusChange,
		Status: &inputlifecycletype.StatusPayload{Key: "processing"},
	})
	res.WaitingForAI = true
	return res, nil
}

var _ chatproc.Executor = ChatActionExecutor{}

// ControlActionExecutor adapts resolved control actions to the legacy UI action channel.
type ControlActionExecutor struct {
	Sink ActionSink
}

// ExecuteControl implements [controlproc.Executor].
func (e ControlActionExecutor) ExecuteControl(action inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error) {
	switch action {
	case inputlifecycletype.ControlCancelProcessing:
		if e.Sink == nil {
			return inputlifecycletype.ProcessResult{}, ErrActionRejected
		}
		if !e.Sink.Send(uivm.UIAction{Kind: uivm.UIActionCancelRequested}) {
			return inputlifecycletype.ProcessResult{}, ErrActionRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case inputlifecycletype.ControlCloseOverlay,
		inputlifecycletype.ControlClearPreInput:
		return inputlifecycletype.ConsumedResult(), nil
	case inputlifecycletype.ControlQuit, inputlifecycletype.ControlInterrupt:
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputQuit,
		}), nil
	default:
		return inputlifecycletype.ProcessResult{}, controlproc.ErrUnknownControlSignal
	}
}

var _ controlproc.Executor = ControlActionExecutor{}
