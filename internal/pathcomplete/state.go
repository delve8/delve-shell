package pathcomplete

import "sync"

// State stores shared dropdown state for filesystem path completion.
type State struct {
	Candidates []string
	Index      int
}

var currentState struct {
	mu    sync.RWMutex
	state State
}

func GetState() State {
	currentState.mu.RLock()
	defer currentState.mu.RUnlock()
	return currentState.state
}

func SetState(state State) {
	currentState.mu.Lock()
	currentState.state = state
	currentState.mu.Unlock()
}

func ResetState() {
	SetState(State{})
}
