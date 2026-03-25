package hostnotify

import "sync"

// Send side for user submit (bridged into hostbus; consumed by hostcontroller).
var (
	submitMu sync.RWMutex
	submitC  chan<- string
)

// SetSubmitChan wires user text submission to the host controller (via hostbus).
func SetSubmitChan(c chan<- string) {
	submitMu.Lock()
	defer submitMu.Unlock()
	submitC = c
}

// Submit sends text to the host controller (blocking when buffer has space). Returns false if unwired.
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
