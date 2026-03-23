package ui

import tea "github.com/charmbracelet/bubbletea"

type slashDispatchEntry struct {
	handle     func(Model) (Model, tea.Cmd)
	clearInput bool
}

func (m Model) clearSlashInput() Model {
	m.Input.SetValue("")
	m.Input.CursorEnd()
	m.SlashSuggestIndex = 0
	return m
}

// dispatchSlashExact routes exact slash commands through a single table-driven path.
// clearInput controls whether the slash input is consumed after execution.
func (m Model) dispatchSlashExact(cmd string) (Model, tea.Cmd, bool) {
	dispatch := map[string]slashDispatchEntry{
		"/help": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.openHelpOverlay(), nil
			},
			clearInput: true,
		},
		"/config llm": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.openConfigLLMOverlay(), nil
			},
			clearInput: true,
		},
		"/config add-skill": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.openAddSkillOverlay("", "", ""), nil
			},
			clearInput: true,
		},
		"/config add-remote": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.openAddRemoteOverlay(true, false), nil
			},
			clearInput: true,
		},
		"/remote on": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.openAddRemoteOverlay(false, true), nil
			},
			clearInput: true,
		},
		"/remote off": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.RemoteOffChan != nil {
					select {
					case mm.RemoteOffChan <- struct{}{}:
					default:
					}
				}
				return mm, nil
			},
			clearInput: true,
		},
		"/cancel": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.WaitingForAI && mm.CancelRequestChan != nil {
					select {
					case mm.CancelRequestChan <- struct{}{}:
					default:
					}
					mm.WaitingForAI = false
				}
				return mm, nil
			},
			clearInput: true,
		},
		"/config update auto-run list": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm.applyConfigAllowlistUpdate(), nil
			},
			clearInput: true,
		},
		"/config reload": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.ConfigUpdatedChan != nil {
					select {
					case mm.ConfigUpdatedChan <- struct{}{}:
					default:
					}
				}
				return mm, nil
			},
			clearInput: true,
		},
		"/reload": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.ConfigUpdatedChan != nil {
					select {
					case mm.ConfigUpdatedChan <- struct{}{}:
					default:
					}
				}
				return mm, nil
			},
			clearInput: true,
		},
		"/q": {
			handle: func(mm Model) (Model, tea.Cmd) {
				return mm, tea.Quit
			},
			clearInput: false,
		},
		"/sh": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.ShellRequestedChan != nil {
					msgs := make([]string, len(mm.Messages))
					copy(msgs, mm.Messages)
					select {
					case mm.ShellRequestedChan <- msgs:
					default:
					}
				}
				return mm, tea.Quit
			},
			clearInput: false,
		},
		"/new": {
			handle: func(mm Model) (Model, tea.Cmd) {
				if mm.SubmitChan != nil {
					mm.SubmitChan <- "/new"
				}
				mm = mm.clearSlashInput()
				mm.Viewport.SetContent(mm.buildContent())
				mm.Viewport.GotoBottom()
				return mm, nil
			},
			clearInput: false,
		},
	}
	entry, ok := dispatch[cmd]
	if !ok {
		return m, nil, false
	}
	m, outCmd := entry.handle(m)
	if entry.clearInput {
		m = m.clearSlashInput()
	}
	return m, outCmd, true
}
