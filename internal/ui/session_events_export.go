package ui

import "delve-shell/internal/history"

// SessionEventsToMessages exposes session history rendering helper to feature packages.
func SessionEventsToMessages(events []history.Event, lang string, width int) []string {
	return sessionEventsToMessages(events, lang, width)
}
