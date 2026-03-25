package interactive

import (
	"os"
	"os/exec"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostcontroller"
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/run"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/session"
	"delve-shell/internal/ui"
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
	WireHostChannels(ports, shellRequestedChan)

	var savedMessages []string
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
	hostnotify.SetAllowlistAutoRunGetter(getAllowlistAutoRun)
	run.SetSyncAllowlistAutoRun(func(v bool) {
		currentAllowlistAutoRun.Store(v)
		runners.SetAllowlistAutoRun(v)
	})

	initialShowConfigLLM := pf.NeedConfigLLM
	for {
		controller.SyncCurrentSessionPath()
		hostnotify.SetOpenConfigLLMOnFirstLayout(initialShowConfigLLM)
		initialShowConfigLLM = false
		model := ui.NewModel(savedMessages)
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus())
		currentP.Store(p)
		_, runErr := p.Run()
		currentP.Store(nil)
		if runErr != nil {
			return runErr
		}
		select {
		case savedMessages = <-shellRequestedChan:
			sh := exec.Command("bash", "-i")
			sh.Stdin = os.Stdin
			sh.Stdout = os.Stdout
			sh.Stderr = os.Stderr
			_ = sh.Run()
		default:
			return nil
		}
	}
}
