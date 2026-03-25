package ui

import (
	"strings"

	"delve-shell/internal/inputbridge"
	"delve-shell/internal/inputlifecycle"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/inputoutput"
	"delve-shell/internal/inputprocess/slashproc"

	tea "github.com/charmbracelet/bubbletea"
)

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

type localSlashExecutor struct{}

func (localSlashExecutor) ExecuteSlash(req slashproc.ExecutionRequest) (inputlifecycletype.ProcessResult, error) {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{Value: LifecycleSlashExecuteMsg{
			RawText:       req.RawText,
			InputLine:     req.InputLine,
			SelectedIndex: req.SelectedIndex,
		}},
	}), nil
}

func (m Model) lifecycleEngine() inputlifecycle.Engine {
	return inputbridge.NewEngine(m.ActionSender, uiControlContexts{m: m}, localSlashExecutor{})
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
		msg, ok := out.Message.Value.(LifecycleSlashExecuteMsg)
		if !ok {
			continue
		}
		return m.handleLifecycleSlashExecuteMsg(msg)
	}
	patch, cmd := inputoutput.ApplyResult(res)
	if patch.WaitingForAI != nil {
		m.Interaction.WaitingForAI = *patch.WaitingForAI
	}
	return m, cmd
}
