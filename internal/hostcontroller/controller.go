package hostcontroller

import (
	"context"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/uipresenter"
)

type Options struct {
	Stop <-chan struct{}

	Bus      *hostbus.Bus
	Inputs   hostbus.InputPorts
	CurrentP *atomic.Pointer[tea.Program]

	Sessions *sessionmgr.Manager
	Runners  *runnermgr.Manager

	Executors *executormgr.Manager
	GetExec   func() execenv.CommandExecutor

	CurrentAllowlistAutoRun *atomic.Bool

	SyncSessionPath func(path string)

	// OnEventDispatch is optional; invoked at the start of each dequeued event before the handler runs.
	// Use hostbus.Event.RedactedSummary for logs (no secrets).
	OnEventDispatch func(e hostbus.Event)
}

// Controller is the single orchestration core for host-side flows.
type Controller struct {
	stop <-chan struct{}

	bus *hostbus.Bus

	ui *uipresenter.Presenter

	currentP *atomic.Pointer[tea.Program]

	sessions *sessionmgr.Manager
	runners  *runnermgr.Manager

	executors *executormgr.Manager
	getExec   func() execenv.CommandExecutor

	currentAllowlistAutoRun *atomic.Bool
	syncSessionPath         func(path string)

	fsm    *hostfsm.Machine
	fsmCtx hostfsm.Context

	llmRunning bool
	llmCancel  context.CancelFunc

	onEventDispatch func(hostbus.Event)
}

func New(opts Options) *Controller {
	c := &Controller{
		stop: opts.Stop,

		bus: opts.Bus,
		ui:  uipresenter.New(uipresenter.BusSender{Bus: opts.Bus}),

		currentP: opts.CurrentP,

		sessions: opts.Sessions,
		runners:  opts.Runners,

		executors: opts.Executors,
		getExec:   opts.GetExec,

		currentAllowlistAutoRun: opts.CurrentAllowlistAutoRun,
		syncSessionPath:         opts.SyncSessionPath,

		fsm: hostfsm.NewMachine(hostfsm.StateIdle),

		onEventDispatch: opts.OnEventDispatch,
	}
	hostbus.BridgeInputs(opts.Stop, opts.Bus, opts.Inputs)
	hostbus.StartUIPump(opts.Stop, opts.Bus, opts.CurrentP)
	return c
}

func (c *Controller) Start() {
	go c.run()
}

func (c *Controller) run() {
	for {
		select {
		case <-c.stop:
			if c.llmRunning && c.llmCancel != nil {
				c.llmCancel()
			}
			return
		case e := <-c.bus.Events():
			c.handleEvent(e)
		}
	}
}

func (c *Controller) handleEvent(e hostbus.Event) {
	if c.onEventDispatch != nil {
		c.onEventDispatch(e)
	}
	h, ok := hostEventHandlers[e.Kind]
	if !ok {
		return
	}
	h(c, e)
}

func (c *Controller) SyncCurrentSessionPath() {
	if c.syncSessionPath == nil {
		return
	}
	if s := c.sessions.Current(); s != nil {
		c.syncSessionPath(s.Path())
	}
}
