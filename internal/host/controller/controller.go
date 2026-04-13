package controller

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/hostmem"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/execcancel"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/ui/presenter"
)

type Options struct {
	Stop <-chan struct{}

	Bus           *bus.Bus
	Inputs        bus.InputPorts
	CurrentP      *atomic.Pointer[tea.Program]
	Commands      <-chan hostcmd.Command
	ShellSnapshot chan<- hostcmd.ShellSnapshot

	Sessions *sessionmgr.Manager
	Runners  *runnermgr.Manager

	Executors *executormgr.Manager
	GetExec   func() execenv.CommandExecutor

	SyncSessionPath func(path string)

	// Runtime mirrors access mode for offline checks (optional; nil skips mirror updates).
	Runtime *app.Runtime

	// OnEventDispatch is optional; invoked at the start of each dequeued event before the handler runs.
	// Use bus.Event.RedactedSummary for logs (no secrets).
	OnEventDispatch func(e bus.Event)

	// ExecCancelHub optional; ESC during [EXECUTING] cancels the in-flight /exec or tool command.
	ExecCancelHub *execcancel.Hub
}

// Controller is the single orchestration core for host-side flows.
type Controller struct {
	stop <-chan struct{}

	bus *bus.Bus

	ui *uipresenter.Presenter

	currentP      *atomic.Pointer[tea.Program]
	commands      <-chan hostcmd.Command
	shellSnapshot chan<- hostcmd.ShellSnapshot

	sessions *sessionmgr.Manager
	runners  *runnermgr.Manager

	executors *executormgr.Manager
	getExec   func() execenv.CommandExecutor

	syncSessionPath func(path string)

	fsm    *hostfsm.Machine
	fsmCtx hostfsm.Context

	llmRunning bool
	llmCancel  context.CancelFunc

	onEventDispatch func(bus.Event)

	runtime *app.Runtime

	execCancelHub *execcancel.Hub

	hostMemoryUpdater *hostmem.BackgroundUpdater
}

func New(opts Options) *Controller {
	c := &Controller{
		stop: opts.Stop,

		bus: opts.Bus,
		ui:  uipresenter.New(uipresenter.BusSender{Bus: opts.Bus}),

		currentP:      opts.CurrentP,
		commands:      opts.Commands,
		shellSnapshot: opts.ShellSnapshot,

		sessions: opts.Sessions,
		runners:  opts.Runners,

		executors:       opts.Executors,
		getExec:         opts.GetExec,
		syncSessionPath: opts.SyncSessionPath,

		fsm: hostfsm.NewMachine(hostfsm.StateIdle),

		onEventDispatch: opts.OnEventDispatch,
		runtime:         opts.Runtime,
		execCancelHub:   opts.ExecCancelHub,
		hostMemoryUpdater: hostmem.NewBackgroundUpdater(hostmem.BackgroundUpdaterOptions{
			Stop:   opts.Stop,
			Logger: slog.Default(),
		}),
	}
	bus.BridgeInputs(opts.Stop, opts.Bus, opts.Inputs)
	bus.StartUIPump(opts.Stop, opts.Bus, opts.CurrentP)
	return c
}

func (c *Controller) Start() {
	go c.run()
	c.bindCurrentSessionHooks()
	c.primeHostMemory("local")
}

func (c *Controller) run() {
	for {
		select {
		case <-c.stop:
			if c.execCancelHub != nil {
				c.execCancelHub.Cancel()
			}
			if c.llmRunning && c.llmCancel != nil {
				c.llmCancel()
			}
			return
		case e := <-c.bus.Events():
			c.handleEvent(e)
		case command := <-c.commands:
			c.handleCommand(command)
		}
	}
}

func (c *Controller) handleEvent(e bus.Event) {
	if c.onEventDispatch != nil {
		c.onEventDispatch(e)
	}
	if traceBusEvents() {
		slog.Info("bus_event", "summary", e.RedactedSummary())
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

// traceBusEvents reports whether to log each dequeued bus event (via [bus.Event.RedactedSummary]).
// Set environment variable DELVE_SHELL_TRACE_BUS to 1 or true.
func traceBusEvents() bool {
	v := strings.TrimSpace(os.Getenv("DELVE_SHELL_TRACE_BUS"))
	return v == "1" || strings.EqualFold(v, "true")
}
