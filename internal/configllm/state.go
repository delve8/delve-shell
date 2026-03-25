package configllm

import (
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
)

// overlayState holds `/config llm` interactive overlay (same lifetime pattern as internal/remote overlay state).
type overlayState struct {
	Active           bool
	Checking         bool
	Error            string
	FieldIndex       int
	BaseURLInput     textinput.Model
	ApiKeyInput      textinput.Model
	ModelInput       textinput.Model
	MaxMessagesInput textinput.Model
	MaxCharsInput    textinput.Model
}

var global struct {
	mu sync.Mutex
	st overlayState
}

func getOverlayState() overlayState {
	global.mu.Lock()
	defer global.mu.Unlock()
	return global.st
}

func setOverlayState(st overlayState) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.st = st
}

// ResetOnOverlayClose clears feature flags when the generic overlay chrome is dismissed (Esc).
func ResetOnOverlayClose() {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.st.Active = false
	global.st.Checking = false
	global.st.Error = ""
}

// OverlayActive reports whether the Config LLM overlay is the active feature body (for tests).
func OverlayActive() bool {
	global.mu.Lock()
	defer global.mu.Unlock()
	return global.st.Active
}
