package interactive

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/controller"
	"delve-shell/internal/host/wiring"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
)

// hostStack wires bus, controller, runner, executor, and app.Runtime for one interactive session.
type hostStack struct {
	controller *controller.Controller
	rt         *app.Runtime
	currentP   *atomic.Pointer[tea.Program]
	shellSnap  <-chan []string
}

// wireHostStack builds runners, host bus ports, controller, and *app.Runtime. Caller must Start() the controller.
func wireHostStack(
	stop <-chan struct{},
	pf *PreflightResult,
	sessions *sessionmgr.Manager,
	syncSessionPath func(string),
) *hostStack {
	hostBus := bus.New(512)
	ports := bus.NewInputPorts()

	var currentAllowlistAutoRun atomic.Bool
	currentAllowlistAutoRun.Store(true)
	if pf.Config != nil {
		currentAllowlistAutoRun.Store(pf.Config.AllowlistAutoRunResolved())
	}

	executors := executormgr.New()
	getExecutor := func() execenv.CommandExecutor { return executors.Get() }

	runners := runnermgr.New(runnermgr.Options{
		RulesText: pf.RulesText,
		LoadConfig: func() (*config.Config, error) {
			return config.LoadEnsured()
		},
		LoadAllowlist: func() ([]config.AllowlistEntry, error) {
			return config.LoadAllowlist()
		},
		LoadSensitivePatterns: func() ([]string, error) {
			return config.LoadSensitivePatterns()
		},
		SessionProvider:  func() *history.Session { return sessions.Current() },
		ExecutorProvider: getExecutor,
		AllowlistAutoRun: currentAllowlistAutoRun.Load(),
		UIEvents:         ports.AgentUIChan,
	})

	shellRequestedChan := make(chan []string, 1)
	rt := app.NewRuntime()
	wiring.BindSendPorts(rt, ports, shellRequestedChan)

	var currentP atomic.Pointer[tea.Program]
	controller := controller.New(controller.Options{
		Stop:                    stop,
		Bus:                     hostBus,
		Inputs:                  ports,
		CurrentP:                &currentP,
		Sessions:                sessions,
		Runners:                 runners,
		Executors:               executors,
		GetExec:                 getExecutor,
		CurrentAllowlistAutoRun: &currentAllowlistAutoRun,
		SyncSessionPath:         syncSessionPath,
	})
	controller.Start()

	getAllowlistAutoRun := func() bool { return currentAllowlistAutoRun.Load() }
	wiring.BindAllowlistAutoRun(rt, getAllowlistAutoRun, func(v bool) {
		currentAllowlistAutoRun.Store(v)
		runners.SetAllowlistAutoRun(v)
	})

	return &hostStack{
		controller: controller,
		rt:         rt,
		currentP:   &currentP,
		shellSnap:  shellRequestedChan,
	}
}
