// Package uitypes holds small UI-facing value types shared across ui, uiregistry, and feature packages.
package uitypes

// SlashOption is one row in the slash command list (command + description).
type SlashOption struct {
	Cmd  string
	Desc string
}

// SlashRunUsageOption is the Cmd string for the /run usage row in slash suggestions (fill-only on select).
const SlashRunUsageOption = "/run <cmd>"
