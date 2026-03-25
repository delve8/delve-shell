package hostnotify

import "delve-shell/internal/hostapp"

// Submit sends user text to the host controller (blocking when buffer has space). Returns false if unwired.
func Submit(text string) bool {
	return hostapp.Submit(text)
}

// TrySubmitNonBlocking sends without blocking; returns false if unwired or buffer full.
func TrySubmitNonBlocking(text string) bool {
	return hostapp.TrySubmitNonBlocking(text)
}
