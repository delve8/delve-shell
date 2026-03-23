package ui

import tea "github.com/charmbracelet/bubbletea"

// slashDispatchEntry defines an exact slash command handler.
// The registry is populated via init() functions in feature files.
type slashDispatchEntry struct {
	handle     func(Model) (Model, tea.Cmd)
	clearInput bool
}

var slashExactDispatchRegistry = map[string]slashDispatchEntry{}

func registerSlashExact(cmd string, entry slashDispatchEntry) {
	if cmd == "" {
		return
	}
	if _, ok := slashExactDispatchRegistry[cmd]; ok {
		panic("duplicate exact slash registration: " + cmd)
	}
	slashExactDispatchRegistry[cmd] = entry
}
