package hostloop

import (
	"delve-shell/internal/agent"
	"delve-shell/internal/ui"
)

// agentUIDispatch is a linear table: first Match that returns true wins.
type agentUIDispatch struct {
	Match func(any) bool
	Do    func(d *Deps, x any)
}

var agentUITable []agentUIDispatch

// RegisterAgentUI registers one handler for agent→TUI payloads on UIEvents. Call from init().
func RegisterAgentUI(match func(any) bool, do func(d *Deps, x any)) {
	agentUITable = append(agentUITable, agentUIDispatch{Match: match, Do: do})
}

func dispatchAgentUI(d *Deps, x any) {
	for i := range agentUITable {
		if agentUITable[i].Match(x) {
			agentUITable[i].Do(d, x)
			return
		}
	}
}

func init() {
	RegisterAgentUI(
		func(x any) bool { _, ok := x.(*agent.ApprovalRequest); return ok },
		func(d *Deps, x any) { d.Send(x) },
	)
	RegisterAgentUI(
		func(x any) bool { _, ok := x.(*agent.SensitiveConfirmationRequest); return ok },
		func(d *Deps, x any) { d.Send(x) },
	)
	RegisterAgentUI(
		func(x any) bool { _, ok := x.(agent.ExecEvent); return ok },
		func(d *Deps, x any) {
			v := x.(agent.ExecEvent)
			d.Send(ui.CommandExecutedMsg{
				Command: v.Command, Allowed: v.Allowed, Result: v.Result,
				Sensitive: v.Sensitive, Suggested: v.Suggested,
			})
		},
	)
}
