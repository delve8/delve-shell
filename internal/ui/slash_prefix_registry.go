package ui

import tea "github.com/charmbracelet/bubbletea"
import "delve-shell/internal/slashreg"

// SlashPrefixDispatchEntry routes slash commands with arguments by prefix match.
// Registry is populated by init() in feature packages.
type SlashPrefixDispatchEntry struct {
	Prefix string
	Handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

var slashPrefixDispatchRegistry = slashreg.NewPrefixRegistry[Model, tea.Cmd]()

// RegisterSlashPrefix registers a prefix-based slash command handler.
// Intended to be called from feature packages' init() functions.
func RegisterSlashPrefix(prefix string, entry SlashPrefixDispatchEntry) {
	if prefix == "" {
		return
	}
	if entry.Prefix == "" {
		entry.Prefix = prefix
	}
	slashPrefixDispatchRegistry.Set(prefix, slashreg.PrefixEntry[Model, tea.Cmd]{
		Prefix: entry.Prefix,
		Handle: entry.Handle,
	})
}
