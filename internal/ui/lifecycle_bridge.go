package ui

import (
	"errors"
	"strings"

	"delve-shell/internal/hostcmd"
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

type slashRuntimeExecutor struct {
	sender CommandSender
}

func (e slashRuntimeExecutor) ExecuteSlash(req slashproc.ExecutionRequest) (inputlifecycletype.ProcessResult, error) {
	trimmed := strings.TrimSpace(req.RawText)
	for _, p := range slashExecutionProviderChain.List() {
		if res, handled, err := p(SlashExecutionRequest{
			RawText:       trimmed,
			InputLine:     req.InputLine,
			SelectedIndex: req.SelectedIndex,
			CommandSender: e.sender,
		}); handled {
			return res, err
		}
	}
	switch {
	case trimmed == "/help":
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputOverlayOpen,
			Overlay: &inputlifecycletype.OverlayPayload{
				Title:   i18n.T("en", i18n.KeyHelpTitle),
				Content: i18n.T("en", i18n.KeyHelpText),
			},
		}), nil
	case trimmed == "/new":
		if e.sender == nil || !e.sender.Send(hostcmd.SessionNew{}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case strings.HasPrefix(trimmed, "/session "):
		sessionID := strings.TrimSpace(strings.TrimPrefix(trimmed, "/session "))
		if e.sender == nil || !e.sender.Send(hostcmd.SessionSwitch{SessionID: sessionID}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case strings.HasPrefix(trimmed, "/exec "):
		cmd := strings.TrimSpace(strings.TrimPrefix(trimmed, "/exec "))
		if cmd == "" {
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputTranscriptAppend,
				Transcript: &inputlifecycletype.TranscriptPayload{
					Lines: []inputlifecycletype.TranscriptLine{
						{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T("en", i18n.KeyUsageRun)},
					},
				},
			}), nil
		}
		if e.sender == nil || !e.sender.Send(hostcmd.ExecDirect{Command: cmd}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputSlashExecute,
		Slash: &inputlifecycletype.SlashExecutionPayload{
			RawText:       req.RawText,
			InputLine:     req.InputLine,
			SelectedIndex: req.SelectedIndex,
		},
	}), nil
}

type uiChatSubmissionExecutor struct {
	sender CommandSender
}

func (e uiChatSubmissionExecutor) ExecuteChat(sub inputlifecycletype.InputSubmission) (inputlifecycletype.ProcessResult, error) {
	if e.sender == nil {
		return inputlifecycletype.ProcessResult{}, errUIIntentRejected
	}
	if !e.sender.Send(hostcmd.Submission{Submission: sub}) {
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
	sender CommandSender
}

func (e uiControlActionExecutor) ExecuteControl(action inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error) {
	switch action {
	case inputlifecycletype.ControlCancelProcessing:
		if e.sender == nil {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		if !e.sender.Send(hostcmd.CancelRequested{}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind:   inputlifecycletype.OutputStatusChange,
			Status: &inputlifecycletype.StatusPayload{Key: "idle"},
		}), nil
	case inputlifecycletype.ControlCloseOverlay,
		inputlifecycletype.ControlClearPreInput:
		kind := inputlifecycletype.OutputOverlayClose
		if action == inputlifecycletype.ControlClearPreInput {
			kind = inputlifecycletype.OutputPreInputClear
		}
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: kind,
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
		controlproc.New(uiControlContexts{m: m}, uiControlActionExecutor{sender: m.CommandSender}),
		slashproc.New(slashRuntimeExecutor{sender: m.CommandSender}),
		chatproc.New(uiChatSubmissionExecutor{sender: m.CommandSender}),
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
		switch out.Kind {
		case inputlifecycletype.OutputTranscriptAppend:
			if out.Transcript != nil {
				lines := make([]uivm.Line, 0, len(out.Transcript.Lines))
				for _, line := range out.Transcript.Lines {
					lines = append(lines, uivm.Line{Kind: uivm.LineKind(line.Kind), Text: line.Text})
				}
				return m.handleTranscriptAppendMsg(TranscriptAppendMsg{Lines: lines})
			}
		case inputlifecycletype.OutputSlashExecute:
			if out.Slash != nil {
				if out.Slash.InputLine != "" {
					m2, cmd, handled := m.executeSlashEarlySubmission(out.Slash.InputLine)
					if handled {
						return m2, cmd
					}
					return m.executeSlashSubmission(out.Slash.InputLine, out.Slash.SelectedIndex)
				}
				return m.executeSlashSubmission(out.Slash.RawText, out.Slash.SelectedIndex)
			}
		case inputlifecycletype.OutputOverlayOpen:
			if out.Overlay != nil {
				req := OverlayOpenRequest{
					Key:     out.Overlay.Key,
					Params:  out.Overlay.Params,
					Title:   out.Overlay.Title,
					Content: out.Overlay.Content,
				}
				for _, entry := range overlayFeatures() {
					if entry.feature.Open == nil {
						continue
					}
					if m2, cmd, handled := entry.feature.Open(m, req); handled {
						return m2, cmd
					}
				}
				if req.Title != "" || req.Content != "" {
					m = m.OpenOverlayFeature("", req.Title, req.Content)
					m = m.InitOverlayViewport()
					return m, nil
				}
				return m, nil
			}
		case inputlifecycletype.OutputOverlayClose:
			// Restore focus to the main input after dismissing the overlay (Esc / same as handleOverlayKey).
			return m.closeOverlayCommon(true)
		case inputlifecycletype.OutputPreInputClear:
			return m.clearSlashInput(), nil
		}
	}
	patch, cmd := inputoutput.ApplyResult(res)
	if patch.WaitingForAI != nil {
		m.Interaction.WaitingForAI = *patch.WaitingForAI
	}
	return m, cmd
}
