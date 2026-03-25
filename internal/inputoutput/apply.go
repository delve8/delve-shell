package inputoutput

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/inputlifecycletype"
)

// StatePatch is the UI-agnostic effect summary produced from lifecycle outputs.
type StatePatch struct {
	WaitingForAI *bool
	Quit         bool
}

// ApplyResult summarizes the result into a minimal patch plus any terminal command.
func ApplyResult(res inputlifecycletype.ProcessResult) (StatePatch, tea.Cmd) {
	var patch StatePatch
	if res.WaitingForAI {
		v := true
		patch.WaitingForAI = &v
	}

	var cmds []tea.Cmd
	for _, out := range res.Outputs {
		switch out.Kind {
		case inputlifecycletype.OutputStatusChange:
			if out.Status != nil && out.Status.Key == "processing" {
				v := true
				patch.WaitingForAI = &v
			}
		case inputlifecycletype.OutputMessage:
			if out.Message != nil && out.Message.Value != nil {
				msg := out.Message.Value
				cmds = append(cmds, func() tea.Msg { return msg })
			}
		case inputlifecycletype.OutputQuit:
			patch.Quit = true
			cmds = append(cmds, tea.Quit)
		}
	}
	return patch, tea.Batch(cmds...)
}
