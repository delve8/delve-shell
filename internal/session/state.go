package session

import "sync"

var currentSessionState struct {
	mu   sync.RWMutex
	path string
}

func setCurrentSessionPath(path string) {
	currentSessionState.mu.Lock()
	currentSessionState.path = path
	currentSessionState.mu.Unlock()
}

func getCurrentSessionPath() string {
	currentSessionState.mu.RLock()
	defer currentSessionState.mu.RUnlock()
	return currentSessionState.path
}

// SetCurrentSessionPath is used by host loop to update session module state.
func SetCurrentSessionPath(path string) {
	setCurrentSessionPath(path)
}
