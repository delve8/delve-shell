package ui

import (
	"errors"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycle"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputoutput"
	"delve-shell/internal/inputpreflight"
	"delve-shell/internal/inputprocess/chatproc"
	"delve-shell/internal/inputprocess/controlproc"
	"delve-shell/internal/inputprocess/slashproc"
	"delve-shell/internal/uivm"

	tea "github.com/charmbracelet/bubbletea"
)

var errUIIntentRejected = errors.New("ui lifecycle: outbound submission rejected")

type uiControlContexts struct {
	m Model
}

func (c uiControlContexts) ControlContext() inputlifecycletype.ControlContext {
	return inputlifecycletype.ControlContext{
		HasActiveOverlay: c.m.Overlay.Active,
		HasPreInputState: hasSlashPreInputState(c.m),
		WaitingForAI:     c.m.Interaction.WaitingForAI,
	}
}

func hasSlashPreInputState(m Model) bool {
	inputVal := m.Input.Value()
	return inputVal != "" && inputVal[0] == '/'
}

type localSlashExecutor struct {
	sender ActionSender
}

func (e localSlashExecutor) ExecuteSlash(req slashproc.ExecutionRequest) (inputlifecycletype.ProcessResult, error) {
	trimmed := strings.TrimSpace(req.RawText)
	for _, p := range slashExecutionProviderChain.List() {
		if res, handled, err := p(SlashExecutionRequest{
			RawText:       trimmed,
			InputLine:     req.InputLine,
			SelectedIndex: req.SelectedIndex,
			ActionSender:  e.sender,
		}); handled {
			return res, err
		}
	}
	switch {
	case trimmed == "/help":
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputMessage,
			Message: &inputlifecycletype.MessagePayload{Value: OverlayShowMsg{
				Title:   i18n.T("en", i18n.KeyHelpTitle),
				Content: i18n.T("en", i18n.KeyHelpText),
			}},
		}), nil
	case trimmed == "/new":
		if e.sender == nil || !e.sender.Send(uivm.UIAction{Kind: uivm.UIActionSessionNew}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case strings.HasPrefix(trimmed, "/sessions "):
		sessionID := strings.TrimSpace(strings.TrimPrefix(trimmed, "/sessions "))
		if e.sender == nil || !e.sender.Send(uivm.UIAction{Kind: uivm.UIActionSessionSwitch, Text: sessionID}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case strings.HasPrefix(trimmed, "/run "):
		cmd := strings.TrimSpace(strings.TrimPrefix(trimmed, "/run "))
		if cmd == "" {
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputMessage,
				Message: &inputlifecycletype.MessagePayload{Value: TranscriptAppendMsg{
					Lines: []uivm.Line{
						{Kind: uivm.LineSystemError, Text: i18n.T("en", i18n.KeyUsageRun)},
					},
				}},
			}), nil
		}
		if e.sender == nil || !e.sender.Send(uivm.UIAction{Kind: uivm.UIActionExecDirect, Text: cmd}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{Value: LifecycleSlashExecuteMsg{
			RawText:       req.RawText,
			InputLine:     req.InputLine,
			SelectedIndex: req.SelectedIndex,
		}},
	}), nil
}

type uiChatSubmissionExecutor struct {
	sender ActionSender
}

func (e uiChatSubmissionExecutor) ExecuteChat(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	if e.sender == nil {
		return inputlifecycletype.ProcessResult{}, errUIIntentRejected
	}
	if !e.sender.Send(uivm.UIAction{
		Kind:       uivm.UIActionSubmission,
		Submission: sub,
	}) {
		return inputlifecycletype.ProcessResult{}, errUIIntentRejected
	}
	res := inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:   inputlifecycletype.OutputStatusChange,
		Status: &inputlifecycletype.StatusPayload{Key: "processing"},
	})
	res.WaitingForAI = true
	return res, nil
}

type uiControlActionExecutor struct {
	sender ActionSender
}

func (e uiControlActionExecutor) ExecuteControl(action inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error) {
	switch action {
	case inputlifecycletype.ControlCancelProcessing:
		if e.sender == nil {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		if !e.sender.Send(uivm.UIAction{Kind: uivm.UIActionCancelRequested}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind:   inputlifecycletype.OutputStatusChange,
			Status: &inputlifecycletype.StatusPayload{Key: "idle"},
		}), nil
	case inputlifecycletype.ControlCloseOverlay,
		inputlifecycletype.ControlClearPreInput:
		var msg any = OverlayCloseMsg{}
		if action == inputlifecycletype.ControlClearPreInput {
			msg = PreInputClearMsg{}
		}
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputMessage,
			Message: &inputlifecycletype.MessagePayload{
				Value: msg,
			},
		}), nil
	case inputlifecycletype.ControlQuit, inputlifecycletype.ControlInterrupt:
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputQuit,
		}), nil
	default:
		return inputlifecycletype.ProcessResult{}, controlproc.ErrUnknownControlSignal
	}
}

func (m Model) lifecycleEngine() inputlifecycle.Engine {
	router := inputlifecycle.NewRouter(
		controlproc.New(uiControlContexts{m: m}, uiControlActionExecutor{sender: m.ActionSender}),
		slashproc.New(localSlashExecutor{sender: m.ActionSender}),
		chatproc.New(uiChatSubmissionExecutor{sender: m.ActionSender}),
	)
	return inputlifecycle.NewEngine(inputpreflight.Engine{}, router)
}

func (m Model) submitLifecycleSlash(rawText, inputLine string, selectedIndex int, source inputlifecycletype.SubmissionSource) (inputlifecycletype.ProcessResult, bool, error) {
	trimmed := strings.TrimSpace(rawText)
	if trimmed == "" {
		return inputlifecycletype.ProcessResult{}, false, nil
	}
	return m.lifecycleEngine().RouteSubmission(inputlifecycletype.InputSubmission{
		Kind:          inputlifecycletype.SubmissionSlash,
		Source:        source,
		RawText:       trimmed,
		InputLine:     inputLine,
		SelectedIndex: selectedIndex,
	})
}

func (m Model) applyLifecycleResult(res inputlifecycletype.ProcessResult) (Model, tea.Cmd) {
	for _, out := range res.Outputs {
		if out.Kind != inputlifecycletype.OutputMessage || out.Message == nil {
			continue
		}
		msg := out.Message.Value
		switch typed := msg.(type) {
		case LifecycleSlashExecuteMsg:
			return m.handleLifecycleSlashExecuteMsg(typed)
		case OverlayShowMsg:
			return m.handleOverlayShowMsg(typed)
		case OverlayCloseMsg:
			return m.handleOverlayCloseMsg()
		case PreInputClearMsg:
			return m.clearSlashInput(), nil
		case TranscriptAppendMsg:
			return m.handleTranscriptAppendMsg(typed)
		default:
			for _, p := range messageProviderChain.List() {
				if m2, cmd, handled := p(m, msg); handled {
					return m2, cmd
				}
			}
		}
	}
	patch, cmd := inputoutput.ApplyResult(res)
	if patch.WaitingForAI != nil {
		m.Interaction.WaitingForAI = *patch.WaitingForAI
	}
	return m, cmd
}
