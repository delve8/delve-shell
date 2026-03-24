package remote

import (
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
)

// RemoteAuthOverlayState stores overlay-only state for remote authentication prompts.
// Step: "" = inactive, "choose" = selecting auth method, "password" = entering password, "identity" = entering key path.
type RemoteAuthOverlayState struct {
	Step          string
	Target        string
	Error         string
	Username      string          // username to use when submitting (default root)
	UsernameInput textinput.Model // username input in choose step
	Input         textinput.Model // for password or identity path
	Connecting    bool            // true while waiting for remote auth result ("Connecting..." state)
}

// AddRemoteOverlayState stores overlay-only state for add/connect remote dialogs.
type AddRemoteOverlayState struct {
	Active         bool
	UserInput      textinput.Model
	HostInput      textinput.Model
	NameInput      textinput.Model
	KeyInput       textinput.Model
	FieldIndex     int
	Error          string
	OfferOverwrite bool // when true, error was "already exists"; show overwrite hint and accept O to overwrite
	Save           bool // true = save/update remote config; false = only connect (for /remote on)
	Connect        bool // true when opened via /remote on; false for /config add-remote
	Connecting     bool // true while waiting for connection result (show "Connecting...")
}

type remoteOverlayState struct {
	AddRemote  AddRemoteOverlayState
	RemoteAuth RemoteAuthOverlayState
}

var currentRemoteOverlayState struct {
	mu    sync.RWMutex
	state remoteOverlayState
}

func getRemoteOverlayState() remoteOverlayState {
	currentRemoteOverlayState.mu.RLock()
	defer currentRemoteOverlayState.mu.RUnlock()
	return currentRemoteOverlayState.state
}

func setRemoteOverlayState(state remoteOverlayState) {
	currentRemoteOverlayState.mu.Lock()
	currentRemoteOverlayState.state = state
	currentRemoteOverlayState.mu.Unlock()
}

func resetRemoteOverlayState() {
	setRemoteOverlayState(remoteOverlayState{})
}
