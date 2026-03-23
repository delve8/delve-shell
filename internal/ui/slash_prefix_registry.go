package ui

import tea "github.com/charmbracelet/bubbletea"

// SlashPrefixDispatchEntry routes slash commands with arguments by prefix match.
// Registry is populated by init() in feature packages.
type SlashPrefixDispatchEntry struct {
	Prefix string
	Handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

var slashPrefixDispatchRegistry []SlashPrefixDispatchEntry

// RegisterSlashPrefix registers a prefix-based slash command handler.
// Intended to be called from feature packages' init() functions.
func RegisterSlashPrefix(prefix string, entry SlashPrefixDispatchEntry) {
	if prefix == "" {
		return
	}
	if entry.Prefix == "" {
		entry.Prefix = prefix
	}
	for _, e := range slashPrefixDispatchRegistry {
		if e.Prefix == entry.Prefix {
			// Allow overwrite so unit tests can keep default ui registrations
			// while feature packages progressively migrate handlers.
			for i := range slashPrefixDispatchRegistry {
				if slashPrefixDispatchRegistry[i].Prefix == entry.Prefix {
					slashPrefixDispatchRegistry[i] = entry
					return
				}
			}
		}
	}
	slashPrefixDispatchRegistry = append(slashPrefixDispatchRegistry, entry)
}

// registerSlashPrefix is kept for internal callers during incremental refactors.
func registerSlashPrefix(prefix string, entry SlashPrefixDispatchEntry) {
	RegisterSlashPrefix(prefix, entry)
}
