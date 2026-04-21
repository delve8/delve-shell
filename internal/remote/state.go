package remote

import (
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
)

// RemoteAuthOverlayState stores overlay-only state for remote authentication prompts.
// Step: "" = inactive; otherwise AuthStep* constants (choose/password/identity/...).
type RemoteAuthOverlayState struct {
	Step          string
	Target        string
	Socks5Addr    string
	Error         string
	ChoiceIndex   int // 0-based selection for two-choice auth steps (host-key trust, password vs key)
	HostKeyHost   string
	HostKeyFP     string
	Username      string          // username to use when submitting
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
	Socks5Input    textinput.Model
	FieldIndex     int
	Error          string
	ChoiceIndex    int  // 0-based selection for overwrite confirmation choices
	OfferOverwrite bool // when true, error was "already exists"; show overwrite choices
	Save           bool // true = save/update remote config before connect (for /access New overlay)
	Connecting     bool // true while waiting for connection result (show "Connecting...")
}

type RemoteConnectOverlayState struct {
	Active     bool
	Target     string
	Error      string
	Connecting bool
}

type remoteOverlayState struct {
	AddRemote     AddRemoteOverlayState
	ConnectRemote RemoteConnectOverlayState
	RemoteAuth    RemoteAuthOverlayState
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
