package ui

import (
	"errors"
	"strings"

	"delve-shell/internal/history"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycle"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/input/maininput"
	"delve-shell/internal/input/output"
	"delve-shell/internal/input/preflight"
	"delve-shell/internal/input/process/chatproc"
	"delve-shell/internal/input/process/controlproc"
	"delve-shell/internal/input/process/slashproc"
	"delve-shell/internal/slash/view"
	"delve-shell/internal/ui/flow/enterflow"

	"delve-shell/internal/ui/uivm"
	tea "github.com/charmbracelet/bubbletea"
)

var errUIIntentRejected = errors.New("ui lifecycle: outbound submission rejected")

type uiControlContexts struct {
	m *Model
}

func (c uiControlContexts) ControlContext() inputlifecycletype.ControlContext {
	return inputlifecycletype.ControlContext{
		HasActiveOverlay: c.m.Overlay.Active,
		HasPreInputState: hasSlashPreInputState(c.m),
		CommandExecuting: c.m.Interaction.CommandExecuting,
		WaitingForAI:     c.m.Interaction.WaitingForAI,
	}
}

func hasSlashPreInputState(m *Model) bool {
	inputVal := m.Input.Value()
	return inputVal != "" && inputVal[0] == '/'
}

type slashExecutor struct {
	sender            CommandSender
	read              ReadModel
	suggestionContext func(input string) ([]int, []slashview.Option)
	sessionNoneMsg    string
	delRemoteNoneMsg  string
	transcriptLines   func() []string
	inputHistory      func() []string
	remoteActive      func() bool
}

func (e slashExecutor) ExecuteSlash(req slashproc.ExecutionRequest) (inputlifecycletype.ProcessResult, error) {
	trimmed := strings.TrimSpace(req.RawText)
	offline := false
	if e.read != nil {
		offline = e.read.OfflineExecutionMode()
	}
	var selected slashview.Option
	selected.Cmd = strings.TrimSpace(req.SelectedCmd)
	selected.FillValue = strings.TrimSpace(req.SelectedFill)
	if selected.Cmd == "" && e.suggestionContext != nil && req.SelectedIndex >= 0 {
		inputLine := strings.TrimSpace(req.InputLine)
		if inputLine == "" {
			inputLine = trimmed
		}
		vis, viewOpts := e.suggestionContext(inputLine)
		if opt, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, req.SelectedIndex); ok {
			selected = opt
		}
	}
	for _, p := range slashExecutionProviderChain.List() {
		if res, handled, err := p(SlashExecutionRequest{
			RawText:              trimmed,
			InputLine:            req.InputLine,
			SelectedIndex:        req.SelectedIndex,
			SelectedCmd:          selected.Cmd,
			SelectedFill:         selected.FillValue,
			CommandSender:        e.sender,
			OfflineExecutionMode: offline,
		}); handled {
			return res, err
		}
	}
	switch {
	case trimmed == "/help":
		return SlashOverlayOpenResult("", i18n.T(i18n.KeyHelpTitle), i18n.T(i18n.KeyHelpText), true, nil), nil
	case trimmed == "/quit":
		return SlashQuitResult(), nil
	case trimmed == "/new":
		if !SlashTryHostIntent(e.sender, hostcmd.SessionNew{}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	case trimmed == "/bash":
		if offline {
			return SlashTranscriptErrorResult(i18n.T(i18n.KeyOfflineExecBashDisabled)), nil
		}
		mode := hostcmd.SubshellModeLocalBash
		if e.remoteActive != nil && e.remoteActive() {
			mode = hostcmd.SubshellModeRemoteSSH
		}
		if e.sender != nil {
			var msgs []string
			if e.transcriptLines != nil {
				msgs = append([]string(nil), e.transcriptLines()...)
			}
			var hist []string
			if e.inputHistory != nil {
				hist = append([]string(nil), e.inputHistory()...)
			}
			_ = e.sender.Send(hostcmd.ShellSnapshot{Messages: msgs, InputHistory: hist, Mode: mode})
		}
		return SlashQuitResult(), nil
	case strings.HasPrefix(trimmed, "/exec "):
		cmd := strings.TrimSpace(strings.TrimPrefix(trimmed, "/exec "))
		if cmd == "" {
			return SlashTranscriptErrorResult(i18n.T(i18n.KeyUsageRun)), nil
		}
		if e.read != nil && e.read.OfflineExecutionMode() {
			return SlashTranscriptErrorResult(i18n.T(i18n.KeyOfflineSlashExecDisabled)), nil
		}
		if !SlashTryHostIntent(e.sender, hostcmd.ExecDirect{Command: cmd}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
	default:
		if sessionID, ok := history.SwitchSessionIDFromSlashLine(trimmed); ok {
			if !SlashTryHostIntent(e.sender, hostcmd.HistoryPreviewOpen{SessionID: sessionID}) {
				return inputlifecycletype.ProcessResult{}, errUIIntentRejected
			}
			return inputlifecycletype.ConsumedResult(), nil
		}
	}
	if e.suggestionContext != nil {
		inputLine := strings.TrimSpace(req.InputLine)
		if inputLine == "" {
			inputLine = trimmed
		}
		vis, viewOpts := e.suggestionContext(inputLine)
		plan := enterflow.PlanAfterSlashDispatches(trimmed, req.SelectedIndex, viewOpts, vis, e.sessionNoneMsg, e.delRemoteNoneMsg)
		switch plan.Kind {
		case maininput.MainEnterShowSessionNone:
			return SlashTranscriptSuggestResult(e.sessionNoneMsg), nil
		case maininput.MainEnterShowDelRemoteNone:
			return SlashTranscriptSuggestResult(e.delRemoteNoneMsg), nil
		case maininput.MainEnterResolveSelected:
			return SlashPreInputSetResult(slashview.ChosenToInputValue(plan.Selected)), nil
		case maininput.MainEnterUnknownSlash:
			return SlashTranscriptErrorResult(i18n.T(i18n.KeyUnknownCmd)), nil
		}
	}
	return SlashTranscriptErrorResult(i18n.T(i18n.KeyUnknownCmd)), nil
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
	return SlashProcessingResult(), nil
}

type uiControlActionExecutor struct {
	sender CommandSender
}

func (e uiControlActionExecutor) ExecuteControl(action inputlifecycletype.ControlAction) (inputlifecycletype.ProcessResult, error) {
	switch action {
	case inputlifecycletype.ControlCancelCommandExecution:
		if e.sender == nil {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		if !e.sender.Send(hostcmd.CancelRequested{}) {
			return inputlifecycletype.ProcessResult{}, errUIIntentRejected
		}
		return inputlifecycletype.ConsumedResult(), nil
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

func (m *Model) lifecycleEngine() inputlifecycle.Engine {
	router := inputlifecycle.NewRouter(
		controlproc.New(uiControlContexts{m: m}, uiControlActionExecutor{sender: m.CommandSender}),
		slashproc.New(slashExecutor{
			sender: m.CommandSender,
			read:   m.ReadModel,
			suggestionContext: func(input string) ([]int, []slashview.Option) {
				_, vis, viewOpts := m.slashSuggestionContext(input)
				return vis, viewOpts
			},
			sessionNoneMsg:   i18n.T(i18n.KeySessionNone),
			delRemoteNoneMsg: i18n.T(i18n.KeyDelRemoteNoHosts),
			transcriptLines:  m.TranscriptLines,
			inputHistory: func() []string {
				out := make([]string, len(m.Interaction.inputHistory))
				copy(out, m.Interaction.inputHistory)
				return out
			},
			remoteActive: func() bool { return m.Remote.Active },
		}),
		chatproc.New(uiChatSubmissionExecutor{sender: m.CommandSender}),
	)
	return inputlifecycle.NewEngine(inputpreflight.Engine{}, router)
}

func (m *Model) applyLifecycleResult(res inputlifecycletype.ProcessResult) (*Model, tea.Cmd) {
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
		case inputlifecycletype.OutputOverlayOpen:
			if out.Overlay != nil {
				req := OverlayOpenRequest{
					Key:      out.Overlay.Key,
					Params:   out.Overlay.Params,
					Title:    out.Overlay.Title,
					Content:  out.Overlay.Content,
					Markdown: out.Overlay.Markdown,
				}
				for _, entry := range overlayFeatures() {
					if entry.feature.Open == nil {
						continue
					}
					if m2, cmd, handled := entry.feature.Open(m, req); handled {
						return m2, cmd
					}
				}
				if req.Markdown && strings.TrimSpace(req.Content) != "" {
					i18n.SetLang(m.getLang())
					m.openMarkdownScrollOverlay(req.Title, req.Content, i18n.T(i18n.KeyHelpOverlayFooter))
					return m, nil
				}
				if req.Title != "" || req.Content != "" {
					m.OpenOverlayFeature("", req.Title, req.Content)
					m.InitOverlayViewport()
					return m, nil
				}
				return m, nil
			}
		case inputlifecycletype.OutputOverlayClose:
			// Restore focus to the main input after dismissing the overlay (Esc / same as handleOverlayKey).
			return m.closeOverlayCommon(true)
		case inputlifecycletype.OutputPreInputClear:
			m.clearSlashInput()
			return m, nil
		case inputlifecycletype.OutputPreInputSet:
			if out.PreInput != nil {
				m.Input.SetValue(out.PreInput.Value)
				m.Input.CursorEnd()
				m.syncInputHeight()
				return m, nil
			}
		}
	}
	patch, cmd := inputoutput.ApplyResult(res)
	if patch.WaitingForAI != nil {
		m.Interaction.WaitingForAI = *patch.WaitingForAI
	}
	if patch.CommandExecuting != nil {
		m.Interaction.CommandExecuting = *patch.CommandExecuting
	}
	return m, cmd
}
