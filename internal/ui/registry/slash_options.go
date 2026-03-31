// Package uiregistry holds provider chains that do not depend on ui.Model (slash option lists).
package uiregistry

import (
	"delve-shell/internal/i18n"
	"delve-shell/internal/slash/reg"
)

// SlashOption is one row in the slash command list (command + description).
type SlashOption struct {
	Cmd       string
	Desc      string
	FillValue string
}

// SlashOptionsProvider supplies slash dropdown rows for a given input buffer.
// When handled==true, the returned options should override the default root list.
type SlashOptionsProvider func(
	inputVal string,
	lang string,
) (opts []SlashOption, handled bool)

var slashOptionsProviderChain = slashreg.NewProviderChain[SlashOptionsProvider]()

// RootSlashOptionProvider supplies top-level / commands for the root dropdown.
type RootSlashOptionProvider func(lang string) []SlashOption

var rootSlashOptionProviderChain = slashreg.NewProviderChain[RootSlashOptionProvider]()

// RegisterSlashOptionsProvider registers a slash options provider.
func RegisterSlashOptionsProvider(p SlashOptionsProvider) {
	if p == nil {
		return
	}
	slashOptionsProviderChain.Add(p, func(x SlashOptionsProvider) bool { return x == nil })
}

// RegisterRootSlashOptionProvider registers a provider for top-level slash options (concatenated in order).
func RegisterRootSlashOptionProvider(p RootSlashOptionProvider) {
	if p == nil {
		return
	}
	rootSlashOptionProviderChain.Add(p, func(x RootSlashOptionProvider) bool { return x == nil })
}

// RootSlashOptions returns merged top-level slash rows from registered root providers.
func RootSlashOptions(lang string) []SlashOption {
	i18n.SetLang(lang)
	opts := make([]SlashOption, 0, 16)
	for _, p := range rootSlashOptionProviderChain.List() {
		opts = append(opts, p(lang)...)
	}
	return opts
}

// SlashOptionsForInput returns slash options for the current input buffer.
func SlashOptionsForInput(inputVal, lang string) []SlashOption {
	i18n.SetLang(lang)
	for _, p := range slashOptionsProviderChain.List() {
		if o, handled := p(inputVal, lang); handled {
			return o
		}
	}
	return RootSlashOptions(lang)
}
