package ui

import tea "github.com/charmbracelet/bubbletea"

// slashPrefixDispatchEntry routes slash commands with arguments by prefix match.
// Registry is populated by init() in feature files.
type slashPrefixDispatchEntry struct {
	prefix string
	handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

var slashPrefixDispatchRegistry []slashPrefixDispatchEntry

func registerSlashPrefix(prefix string, entry slashPrefixDispatchEntry) {
	if prefix == "" {
		return
	}
	if entry.prefix == "" {
		entry.prefix = prefix
	}
	for _, e := range slashPrefixDispatchRegistry {
		if e.prefix == entry.prefix {
			panic("duplicate slash prefix registration: " + entry.prefix)
		}
	}
	slashPrefixDispatchRegistry = append(slashPrefixDispatchRegistry, entry)
}
