package interactive

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/hostapp"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostcontroller"
	"delve-shell/internal/hostwiring"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/session"
)

// Run starts the interactive TUI loop, host controller, and optional subshell return path.
func Run() error {
	stop := make(chan struct{})
	defer close(stop)

	pf, err := RunPreflight()
	if err != nil {
		return err
	}

	sessions := sessionmgr.New(pf.InitialSession)
	syncSessionPath := func(path string) { session.SetCurrentSessionPath(path) }
	syncSessionPath(pf.InitialSession.Path())
	defer sessions.CloseAll()

	bus := hostbus.New(512)
	ports := hostbus.NewInputPorts()

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
	rt := hostapp.NewRuntime()
	hostwiring.BindSendPorts(rt, ports, shellRequestedChan)

	var currentP atomic.Pointer[tea.Program]
	controller := hostcontroller.New(hostcontroller.Options{
		Stop:                    stop,
		Bus:                     bus,
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
	hostwiring.BindAllowlistAutoRun(rt, getAllowlistAutoRun, func(v bool) {
		currentAllowlistAutoRun.Store(v)
		runners.SetAllowlistAutoRun(v)
	})

	loop := newTuiRestartLoop(controller, &currentP, shellRequestedChan, pf.NeedConfigLLM, rt)
	return loop.run()
}
