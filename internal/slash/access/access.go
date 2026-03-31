// Package slashaccess defines stable tokens for /access slash commands (display + filter).
package slashaccess

const Prefix = "/access "

// Reserved row suffixes (Title case in UI lists).
const (
	ReservedNew     = "New"
	ReservedLocal   = "Local"
	ReservedOffline = "Offline"
)

// Lowercase tokens used when matching partially typed /access input.
const (
	FilterNew     = "new"
	FilterLocal   = "local"
	FilterOffline = "offline"
)

// Command builds "/access <reserved>".
func Command(reserved string) string {
	return Prefix + reserved
}
