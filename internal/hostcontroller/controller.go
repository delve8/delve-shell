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
	switch e.Kind {
	case hostbus.KindSessionNewRequested:
		c.handleSubmitNewSession()
	case hostbus.KindSessionSwitchRequested:
		c.handleSubmitSwitchSession(e.SessionID)
	case hostbus.KindUserChatSubmitted:
		c.handleUserChat(e.UserText)
	case hostbus.KindConfigUpdated:
		c.handleConfigUpdated()
	case hostbus.KindCancelRequested:
		c.handleCancelRequest()
	case hostbus.KindExecDirectRequested:
		c.handleExecDirect(e.Command)
	case hostbus.KindRemoteOnRequested:
		c.handleRemoteOn(e.RemoteTarget)
	case hostbus.KindRemoteOffRequested:
		c.handleRemoteOff()
	case hostbus.KindRemoteAuthResponseSubmitted:
		c.handleRemoteAuthResp(e.RemoteAuthResponse)
	case hostbus.KindApprovalRequested:
		if e.Approval != nil {
			c.ui.ShowApproval(e.Approval)
		}
	case hostbus.KindSensitiveConfirmationRequested:
		if e.Sensitive != nil {
			c.ui.ShowSensitiveConfirmation(e.Sensitive)
		}
	case hostbus.KindAgentExecEvent:
		v := e.AgentExec
		c.ui.CommandExecutedFromTool(v.Command, v.Allowed, v.Result, v.Sensitive, v.Suggested)
	case hostbus.KindAgentUnknown:
		c.handleAgentUI(e.AgentUI)
	case hostbus.KindLLMRunCompleted:
		c.handleLLMRunCompleted(e.Reply, e.Err)
	}
}

func (c *Controller) SyncCurrentSessionPath() {
	if c.syncSessionPath == nil {
		return
	}
	if s := c.sessions.Current(); s != nil {
		c.syncSessionPath(s.Path())
	}
}
