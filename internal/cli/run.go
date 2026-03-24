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

	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/cli/hostloop"
	"delve-shell/internal/config"
	_ "delve-shell/internal/configllm"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	_ "delve-shell/internal/remote"
	"delve-shell/internal/rules"
	_ "delve-shell/internal/run"
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

	uiEvents := make(chan any, 16)
	configUpdatedChan := make(chan struct{}, 1)
	allowlistAutoRunChangeChan := make(chan bool, 1)

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
		UIEvents:         uiEvents,
	})

	submitChan := make(chan string, 4)
	execDirectChan := make(chan string, 4)
	shellRequestedChan := make(chan []string, 1)
	cancelRequestChan := make(chan struct{}, 1)
	remoteOnChan := make(chan string, 1)
	remoteOffChan := make(chan struct{}, 1)
	remoteAuthRespChan := make(chan ui.RemoteAuthResponse, 1)
	var savedMessages []string
	var currentP atomic.Pointer[tea.Program]
	uiMsgChan := make(chan tea.Msg, 256)

	sendToUI := func(msg tea.Msg) {
		if msg == nil {
			return
		}
		select {
		case uiMsgChan <- msg:
		default:
		}
	}

	deps := &hostloop.Deps{
		Stop:                       stop,
		Send:                       sendToUI,
		Sessions:                   sessions,
		Runners:                    runners,
		Executors:                  executors,
		SyncSessionPath:            syncSessionPath,
		GetExecutor:                getExecutor,
		CurrentP:                   &currentP,
		CurrentAllowlistAutoRun:    &currentAllowlistAutoRun,
		UIEvents:                   uiEvents,
		ConfigUpdatedChan:          configUpdatedChan,
		AllowlistAutoRunChangeChan: allowlistAutoRunChangeChan,
		ExecDirectChan:             execDirectChan,
		RemoteOnChan:               remoteOnChan,
		RemoteOffChan:              remoteOffChan,
		RemoteAuthRespChan:         remoteAuthRespChan,
	}

	fsm := hostfsm.NewMachine(hostfsm.StateIdle)
	hostloop.StartBackgroundLoops(stop, deps, uiMsgChan, submitChan, cancelRequestChan, fsm, &currentP)

	getAllowlistAutoRun := func() bool { return currentAllowlistAutoRun.Load() }
	initialShowConfigLLM := needConfigLLM
	for {
		if s := sessions.Current(); s != nil {
			syncSessionPath(s.Path())
		}
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, allowlistAutoRunChangeChan, remoteOnChan, remoteOffChan, remoteAuthRespChan, getAllowlistAutoRun, savedMessages, initialShowConfigLLM)
		model.Context.ConfigPath = config.ConfigPath()
		initialShowConfigLLM = false
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
