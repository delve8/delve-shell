// Package slashskill defines stable tokens for /skill slash commands (display + filter).
package slashskill

const (
	Subcommand = "skill"
	Prefix     = "/" + Subcommand + " "
)

// Reserved row suffixes (Title case in UI lists).
const (
	ReservedNew    = "New"
	ReservedRemove = "Remove"
	ReservedUpdate = "Update"
)

// Lowercase tokens used when matching partially typed /skill input.
const (
	FilterNew    = "new"
	FilterRemove = "remove"
	FilterUpdate = "update"
)

// Command builds "/skill <reserved>".
func Command(reserved string) string {
	return Prefix + reserved
}
