package ui

import tea "github.com/charmbracelet/bubbletea"

// SlashExactDispatchEntry defines an exact slash command handler.
// The registry is populated via init() functions in feature packages.
type SlashExactDispatchEntry struct {
	Handle     func(Model) (Model, tea.Cmd)
	ClearInput bool
}

var slashExactDispatchRegistry = map[string]SlashExactDispatchEntry{}

// RegisterSlashExact registers an exact slash command handler.
// Intended to be called from feature packages' init() functions.
func RegisterSlashExact(cmd string, entry SlashExactDispatchEntry) {
	if cmd == "" {
		return
	}
	if _, ok := slashExactDispatchRegistry[cmd]; ok {
		panic("duplicate exact slash registration: " + cmd)
	}
	slashExactDispatchRegistry[cmd] = entry
}

// registerSlashExact is kept for internal callers during incremental refactors.
func registerSlashExact(cmd string, entry SlashExactDispatchEntry) {
	RegisterSlashExact(cmd, entry)
}
