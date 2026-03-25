package ui

import tea "github.com/charmbracelet/bubbletea"
import "delve-shell/internal/slashreg"

// SlashExactDispatchEntry defines an exact slash command handler.
// The registry is populated from feature packages via explicit Register() (see bootstrap.Install).
type SlashExactDispatchEntry struct {
	Handle     func(Model) (Model, tea.Cmd)
	ClearInput bool
}

// SlashPrefixDispatchEntry routes slash commands with arguments by prefix match.
// Registry is populated by feature packages' Register() (wired through bootstrap.Install).
type SlashPrefixDispatchEntry struct {
	Prefix string
	Handle func(Model, string) (Model, tea.Cmd, bool) // rest after prefix
}

var slashExactDispatchRegistry = slashreg.NewExactRegistry[Model, tea.Cmd]()
var slashPrefixDispatchRegistry = slashreg.NewPrefixRegistry[Model, tea.Cmd]()

// RegisterSlashExact registers an exact slash command handler.
// Intended to be called from feature packages' Register() functions.
func RegisterSlashExact(cmd string, entry SlashExactDispatchEntry) {
	if cmd == "" {
		return
	}
	if _, ok := slashExactDispatchRegistry.Get(cmd); ok {
		// Overwrite to allow ui-level default registrations to coexist with
		// feature-package registrations during incremental refactors and tests.
	}
	slashExactDispatchRegistry.Set(cmd, slashreg.ExactEntry[Model, tea.Cmd]{
		Handle:     entry.Handle,
		ClearInput: entry.ClearInput,
	})
}

// RegisterSlashPrefix registers a prefix-based slash command handler.
// Intended to be called from feature packages' Register() functions.
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
