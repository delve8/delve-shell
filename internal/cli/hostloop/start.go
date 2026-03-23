package hostloop

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/cli/hostfsm"
)

// StartBackgroundLoops runs: UI message pump, host multiplex (one select), submit loop.
// Three goroutines total for this bridge (plus the main thread and tea runtime).
func StartBackgroundLoops(
	stop <-chan struct{},
	d *Deps,
	uiMsgChan <-chan tea.Msg,
	submitChan <-chan string,
	cancelRequestChan <-chan struct{},
	fsm *hostfsm.Machine,
	currentP *atomic.Pointer[tea.Program],
) {
	go RunUIPump(stop, uiMsgChan, currentP)
	go RunHostMultiplex(stop, d)
	go RunSubmitLoop(stop, d, submitChan, cancelRequestChan, fsm)
}
