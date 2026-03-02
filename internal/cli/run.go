package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/rules"
	"delve-shell/internal/ui"
)

func runRun(cmd *cobra.Command, args []string) error {
	_ = args
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if err := history.Prune(cfg); err != nil {
		log.Printf("[warn] history prune: %v", err)
	}
	rulesText, err := rules.Load()
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}
	sessionID := time.Now().Format("20060102-150405") + "-" + uuid.New().String()[:8]
	session, err := history.NewSession(sessionID)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	approvalChan := make(chan *agent.ApprovalRequest, 4)
	execEventChan := make(chan agent.ExecEvent, 8)
	configUpdatedChan := make(chan struct{}, 1)

	var runner *agent.Runner
	var runnerMu sync.Mutex
	getRunner := func() (*agent.Runner, error) {
		runnerMu.Lock()
		defer runnerMu.Unlock()
		if runner != nil {
			return runner, nil
		}
		cfg2, err := loadConfig()
		if err != nil {
			return nil, err
		}
		_, apiKey, _ := cfg2.LLMResolved()
		if apiKey == "" {
			return nil, agent.ErrLLMNotConfigured
		}
		whitelistEntries, err := config.LoadWhitelist()
		if err != nil {
			return nil, fmt.Errorf("load whitelist: %w", err)
		}
		whitelist := hil.NewWhitelist(whitelistEntries)
		r, err := agent.NewRunner(context.Background(), agent.RunnerOptions{
			Config:        cfg2,
			Whitelist:     whitelist,
			Session:       session,
			RulesText:     rulesText,
			ApprovalChan:  approvalChan,
			ExecEventChan: execEventChan,
		})
		if err != nil {
			return nil, err
		}
		runner = r
		return runner, nil
	}

	submitChan := make(chan string, 4)
	execDirectChan := make(chan string, 4)
	shellRequestedChan := make(chan []string, 1)
	cancelRequestChan := make(chan struct{}, 1)
	var savedMessages []string
	var currentP *tea.Program

	go func() {
		for range configUpdatedChan {
			runnerMu.Lock()
			runner = nil
			runnerMu.Unlock()
			if currentP != nil {
				currentP.Send(ui.ConfigReloadedMsg{})
			}
		}
	}()
	go func() {
		for req := range approvalChan {
			if currentP != nil {
				currentP.Send(req)
			}
		}
	}()
	go func() {
		for ev := range execEventChan {
			if currentP != nil {
				currentP.Send(ui.CommandExecutedMsg{Command: ev.Command, Whitelisted: ev.Whitelisted, Result: ev.Result, Sensitive: ev.Sensitive})
			}
		}
	}()
	go func() {
		for cmd := range execDirectChan {
			c := exec.Command("sh", "-c", cmd)
			var stdout, stderr bytes.Buffer
			c.Stdout = &stdout
			c.Stderr = &stderr
			runErr := c.Run()
			exitCode := 0
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
			result := stdout.String()
			if stderr.Len() > 0 {
				result += "\nstderr:\n" + stderr.String()
			}
			result += "\nexit_code: " + fmt.Sprint(exitCode)
			if runErr != nil && exitCode == 0 {
				result += "\nerror: " + runErr.Error()
			}
			if currentP != nil {
				currentP.Send(ui.CommandExecutedMsg{Command: cmd, Direct: true, Result: result})
			}
		}
	}()
	go func() {
		for userMsg := range submitChan {
			r, err := getRunner()
			if err != nil {
				if currentP != nil {
					currentP.Send(ui.AgentReplyMsg{Err: err})
				}
				continue
			}
			reqCtx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			var reply string
			var runErr error
			go func() {
				defer close(done)
				reply, runErr = r.Run(reqCtx, userMsg)
			}()
			select {
			case <-done:
				cancel()
				if runErr != nil {
					if strings.Contains(runErr.Error(), "404") {
						runErr = errors.Join(runErr, fmt.Errorf("%s", "Hint: For DashScope, ensure LLM_BASE_URL and API Key region match (Beijing vs International). See README for curl test."))
					}
					if currentP != nil {
						currentP.Send(ui.AgentReplyMsg{Err: runErr})
					}
					continue
				}
				_ = session.AppendUserInput(userMsg)
				_ = session.AppendLLMResponse(map[string]string{"reply": reply})
				if currentP != nil {
					currentP.Send(ui.AgentReplyMsg{Reply: reply})
				}
			case <-cancelRequestChan:
				cancel()
				<-done
				if currentP != nil {
					currentP.Send(ui.AgentReplyMsg{Err: runErr})
				}
			}
		}
	}()

	for {
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, savedMessages)
		// 不使用 WithMouse*，以便终端可对文字做鼠标选中复制；滚动请用 Up/Down/PgUp/PgDown
		p := tea.NewProgram(model, tea.WithAltScreen())
		currentP = p
		_, err = p.Run()
		currentP = nil
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
			// loop again with savedMessages to restore content
		default:
			return nil
		}
	}
}

func loadConfig() (*config.Config, error) {
	if err := config.EnsureRootDir(); err != nil {
		return nil, err
	}
	return config.Load()
}
