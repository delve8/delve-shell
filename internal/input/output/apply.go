package inputoutput

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/input/lifecycletype"
)

// StatePatch is the UI-agnostic effect summary produced from lifecycle outputs.
type StatePatch struct {
	WaitingForAI     *bool
	CommandExecuting *bool
	Quit             bool
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
			if out.Status != nil {
				switch out.Status.Key {
				case "processing":
					v := true
					patch.WaitingForAI = &v
				case "idle":
					v := false
					patch.WaitingForAI = &v
				}
			}
		case inputlifecycletype.OutputCommandExecution:
			if out.CommandExec != nil {
				v := out.CommandExec.Active
				patch.CommandExecuting = &v
			}
		case inputlifecycletype.OutputQuit:
			patch.Quit = true
			cmds = append(cmds, tea.Quit)
		}
	}
	return patch, tea.Batch(cmds...)
}
