package cli

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"delve-shell/internal/config"
	_ "delve-shell/internal/configllm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostcontroller"
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/remote"
	"delve-shell/internal/rules"
	"delve-shell/internal/run"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/session"
	_ "delve-shell/internal/skill"
	"delve-shell/internal/ui"
)

func Run(cmd *cobra.Command, args []string) error {
	_ = args

	stop := make(chan struct{})
	defer close(stop)

	if err := config.EnsureRootDir(); err != nil {
		return err
	}
	cfg, _ := config.LoadEnsured()
	needConfigLLM := cfg == nil || strings.TrimSpace(cfg.LLM.Model) == ""

	if cfg != nil {
		if err := history.Prune(cfg); err != nil {
			log.Printf("[warn] history prune: %v", err)
		}
	}
	rulesText, err := rules.Load()
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}
	initialSession, err := history.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	sessions := sessionmgr.New(initialSession)
	syncSessionPath := func(path string) { session.SetCurrentSessionPath(path) }
	syncSessionPath(initialSession.Path())
	defer sessions.CloseAll()

	bus := hostbus.New(512)
	ports := hostbus.NewInputPorts()

	var currentAllowlistAutoRun atomic.Bool
	currentAllowlistAutoRun.Store(true)
	if cfg != nil {
		currentAllowlistAutoRun.Store(cfg.AllowlistAutoRunResolved())
	}

	executors := executormgr.New()
	getExecutor := func() execenv.CommandExecutor { return executors.Get() }

	runners := runnermgr.New(runnermgr.Options{
		RulesText: rulesText,
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

	hostnotify.SetSubmitChan(ports.SubmitChan)
	run.SetExecDirectChan(ports.ExecDirectChan)
	hostnotify.SetConfigUpdatedChan(ports.ConfigUpdatedChan)
	shellRequestedChan := make(chan []string, 1)
	run.SetShellRequestedChan(shellRequestedChan)
	run.SetCancelRequestChan(ports.CancelRequestChan)
	remote.SetRemoteOnTargetChan(ports.RemoteOnChan)
	remote.SetRemoteOffChan(ports.RemoteOffChan)
	remote.SetRemoteAuthRespChan(ports.RemoteAuthRespChan)
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
	initialShowConfigLLM := needConfigLLM
	for {
		controller.SyncCurrentSessionPath()
		hostnotify.SetOpenConfigLLMOnFirstLayout(initialShowConfigLLM)
		initialShowConfigLLM = false
		model := ui.NewModel(savedMessages)
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus())
		currentP.Store(p)
		_, err = p.Run()
		currentP.Store(nil)
		if err != nil {
			return err
		}
		select {
		case savedMessages = <-shellRequestedChan:
			shell := exec.Command("bash", "-i")
			shell.Stdin = os.Stdin
			shell.Stdout = os.Stdout
			shell.Stderr = os.Stderr
			_ = shell.Run()
		default:
			return nil
		}
	}
}
