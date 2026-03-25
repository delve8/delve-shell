package hostnotify

import "sync"

// Send side for hostloop user submit (same instance as RunSubmitLoop / cli/run wiring).
var (
	submitMu sync.RWMutex
	submitC  chan<- string
)

// SetSubmitChan wires user text submission to the host submit loop.
func SetSubmitChan(c chan<- string) {
	submitMu.Lock()
	defer submitMu.Unlock()
	submitC = c
}

// Submit sends text to the host submit loop (blocking when buffer has space). Returns false if unwired.
func Submit(text string) bool {
	submitMu.RLock()
	ch := submitC
	submitMu.RUnlock()
	if ch == nil {
		return false
	}
	ch <- text
	return true
}

// TrySubmitNonBlocking sends without blocking; returns false if unwired or buffer full.
func TrySubmitNonBlocking(text string) bool {
	submitMu.RLock()
	ch := submitC
	submitMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- text:
		return true
	default:
		return false
	}
}
