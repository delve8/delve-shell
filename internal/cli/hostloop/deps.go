package hostloop

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/execenv"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/ui"
)

// Deps is shared by host goroutines (multiplex, submit, remote helpers). Built once in cli.Run.
type Deps struct {
	Stop <-chan struct{}
	Send func(tea.Msg)

	Sessions  *sessionmgr.Manager
	Runners   *runnermgr.Manager
	Executors *executormgr.Manager
	// SyncSessionPath updates session module internal current-session state.
	SyncSessionPath func(path string)
	// GetExecutor returns the current executor (local or remote).
	GetExecutor func() execenv.CommandExecutor
	CurrentP    *atomic.Pointer[tea.Program]
	// CurrentAllowlistAutoRun is updated by config reload and runtime toggle.
	CurrentAllowlistAutoRun *atomic.Bool

	UIEvents                   <-chan any
	ConfigUpdatedChan          <-chan struct{}
	AllowlistAutoRunChangeChan <-chan bool
	ExecDirectChan             <-chan string
	RemoteOnChan               <-chan string
	RemoteOffChan              <-chan struct{}
	RemoteAuthRespChan         <-chan ui.RemoteAuthResponse
}
