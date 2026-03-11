package cli

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/rules"
	"delve-shell/internal/ui"
)

func runRun(cmd *cobra.Command, args []string) error {
	_ = args

	// 首次启动向导：当没有 config.yaml 且未显式关闭向导时，先走交互配置。
	cfg, ranWizard, err := ensureConfigWithWizard()
	if err != nil {
		return err
	}
	if ranWizard {
		// 在进入 TUI 前做一次轻量 LLM 连通性测试，仅输出结果，不中断主流程。
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := testLLMConnection(ctx, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "LLM connectivity test failed: %v\n", err)
		} else {
			fmt.Fprintln(os.Stdout, "LLM connectivity test succeeded.")
		}
	}

	if err := history.Prune(cfg); err != nil {
		log.Printf("[warn] history prune: %v", err)
	}
	rulesText, err := rules.Load()
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}
	sessionID := time.Now().Format("060102-150405") + "-" + randHex2()
	session, err := history.NewSession(sessionID)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	approvalChan := make(chan *agent.ApprovalRequest, 4)
	sensitiveConfirmationChan := make(chan *agent.SensitiveConfirmationRequest, 4)
	execEventChan := make(chan agent.ExecEvent, 8)
	configUpdatedChan := make(chan struct{}, 1)
	allowlistAutoRunChangeChan := make(chan bool, 1)

	cfg0, _ := loadConfig()
	currentAllowlistAutoRun := true
	if cfg0 != nil {
		currentAllowlistAutoRun = cfg0.AllowlistAutoRunResolved()
	}

	var runner *agent.Runner
	var runnerMu sync.Mutex

	// currentExecutor is the active command executor (local or remote).
	var currentExecutor execenv.CommandExecutor = execenv.LocalExecutor{}
	// executorMu protects currentExecutor access from concurrent goroutines.
	var executorMu sync.Mutex

	// getExecutor returns the current executor for the Runner.
	getExecutor := func() execenv.CommandExecutor {
		executorMu.Lock()
		defer executorMu.Unlock()
		return currentExecutor
	}

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
		allowlistEntries, err := config.LoadAllowlist()
		if err != nil {
			return nil, fmt.Errorf("load allowlist: %w", err)
		}
		allowlist := hil.NewAllowlist(allowlistEntries)
		sensitivePatterns, err := config.LoadSensitivePatterns()
		if err != nil {
			return nil, fmt.Errorf("load sensitive patterns: %w", err)
		}
		sensitiveMatcher := hil.NewSensitiveMatcher(sensitivePatterns)
		r, err := agent.NewRunner(context.Background(), agent.RunnerOptions{
			Config:                    cfg2,
			AllowlistAutoRun:          &currentAllowlistAutoRun,
			Allowlist:                 allowlist,
			SensitiveMatcher:          sensitiveMatcher,
			Session:                   session,
			RulesText:                 rulesText,
			ApprovalChan:              approvalChan,
			SensitiveConfirmationChan: sensitiveConfirmationChan,
			ExecEventChan:             execEventChan,
			ExecutorProvider:          getExecutor,
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
	sessionSwitchChan := make(chan string, 1)
	remoteOnChan := make(chan string, 1)
	remoteOffChan := make(chan struct{}, 1)
	remoteAuthRespChan := make(chan ui.RemoteAuthResponse, 1)
	var savedMessages []string
	var currentP *tea.Program

	go func() {
		for path := range sessionSwitchChan {
			oldSession := session
			newSession, err := history.OpenSession(path)
			if err != nil {
				if currentP != nil {
					currentP.Send(ui.AgentReplyMsg{Err: fmt.Errorf("open session: %w", err)})
				}
				continue
			}
			_ = oldSession.Close()
			session = newSession
			runnerMu.Lock()
			runner = nil
			runnerMu.Unlock()
			if currentP != nil {
				currentP.Send(ui.SessionSwitchedMsg{Path: path})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-configUpdatedChan:
				if cfg, err := loadConfig(); err == nil && cfg != nil {
					currentAllowlistAutoRun = cfg.AllowlistAutoRunResolved()
				}
				runnerMu.Lock()
				runner = nil
				runnerMu.Unlock()
				if currentP != nil {
					currentP.Send(ui.ConfigReloadedMsg{})
				}
			case newAutoRun := <-allowlistAutoRunChangeChan:
				currentAllowlistAutoRun = newAutoRun
				runnerMu.Lock()
				runner = nil
				runnerMu.Unlock()
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
		for req := range sensitiveConfirmationChan {
			if currentP != nil {
				currentP.Send(req)
			}
		}
	}()
	go func() {
		for ev := range execEventChan {
			if currentP != nil {
				currentP.Send(ui.CommandExecutedMsg{Command: ev.Command, Allowed: ev.Allowed, Result: ev.Result, Sensitive: ev.Sensitive, Suggested: ev.Suggested})
			}
		}
	}()
	go func() {
		for cmd := range execDirectChan {
			executor := getExecutor()
			stdout, stderrStr, exitCode, runErr := executor.Run(context.Background(), cmd)
			result := stdout
			if stderrStr != "" {
				if result != "" {
					result += "\n"
				}
				result += "stderr:\n" + stderrStr
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
		for target := range remoteOnChan {
			// Resolve target against remotes: allow name, full target, or host-only.
			identityFile := ""
			label := target
			remotes, errRemotes := config.LoadRemotes()
			if errRemotes == nil && len(remotes) > 0 {
				for _, r := range remotes {
					matched := r.Target == target || r.Name == target || config.HostFromTarget(r.Target) == target
					if matched && r.Target != "" {
						target = r.Target
						identityFile = r.IdentityFile
						hostOnly := config.HostFromTarget(target)
						if r.Name != "" {
							label = fmt.Sprintf("%s (%s)", r.Name, hostOnly)
						} else {
							label = hostOnly
						}
						break
					}
				}
			}

			sshExec, _, err := execenv.NewSSHExecutor(target, identityFile)
			if err != nil {
				// Authentication-related errors should trigger interactive auth prompt.
				if currentP != nil {
					msg := fmt.Sprintf("Remote connect failed for %s: %v", config.HostFromTarget(target), err)
					currentP.Send(ui.RemoteAuthPromptMsg{Target: target, Err: msg})
				}
				continue
			}
			if label == "" {
				label = config.HostFromTarget(target)
			}
			executorMu.Lock()
			currentExecutor = sshExec
			executorMu.Unlock()
			if currentP != nil {
				currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
				currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
			}
		}
	}()
	go func() {
		for range remoteOffChan {
			executorMu.Lock()
			// If current executor is SSH, close it.
			if sshExec, ok := currentExecutor.(*execenv.SSHExecutor); ok {
				_ = sshExec.Close()
			}
			currentExecutor = execenv.LocalExecutor{}
			executorMu.Unlock()
			if currentP != nil {
				currentP.Send(ui.RemoteStatusMsg{Active: false, Label: ""})
				currentP.Send(ui.SystemNotifyMsg{Text: "Switched back to local executor."})
			}
		}
	}()
	go func() {
		for resp := range remoteAuthRespChan {
			if resp.Password == "" {
				continue
			}
			// Use username from overlay if set; otherwise keep target as-is (user@host from config).
			targetForSSH := resp.Target
			if resp.Username != "" {
				host := config.HostFromTarget(resp.Target)
				targetForSSH = resp.Username + "@" + host
			}
			var sshExec execenv.CommandExecutor
			var err error
			switch resp.Kind {
			case "identity":
				sshExec, _, err = execenv.NewSSHExecutor(targetForSSH, resp.Password)
			default: // "password"
				sshExec, _, err = execenv.NewSSHExecutorWithPassword(targetForSSH, "", resp.Password)
			}
			// Best effort to clear password string.
			resp.Password = ""
			if err != nil {
				if currentP != nil {
					currentP.Send(ui.RemoteAuthPromptMsg{
						Target: resp.Target,
						Err:   fmt.Sprintf("Auth failed: %v", err),
					})
				}
				continue
			}
			executorMu.Lock()
			currentExecutor = sshExec
			executorMu.Unlock()
			if currentP != nil {
				label := config.HostFromTarget(targetForSSH)
				currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
				currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
			}
		}
	}()
	go func() {
		for userMsg := range submitChan {
			if userMsg == "/new" {
				oldSession := session
				sessionID := time.Now().Format("060102-150405") + "-" + randHex2()
				newSession, err := history.NewSession(sessionID)
				if err != nil {
					if currentP != nil {
						currentP.Send(ui.AgentReplyMsg{Err: fmt.Errorf("new session: %w", err)})
					}
					continue
				}
				_ = oldSession.Close()
				session = newSession
				runnerMu.Lock()
				runner = nil
				runnerMu.Unlock()
				if currentP != nil {
					currentP.Send(ui.SessionSwitchedMsg{Path: session.Path()})
				}
				continue
			}
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

	getAllowlistAutoRun := func() bool { return currentAllowlistAutoRun }
	for {
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, allowlistAutoRunChangeChan, sessionSwitchChan, remoteOnChan, remoteOffChan, remoteAuthRespChan, getAllowlistAutoRun, savedMessages, session.Path())
		// do not use WithMouse* so the terminal can use mouse for text selection; scroll with Up/Down/PgUp/PgDown
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

// randHex2 returns 2 random hex chars from crypto/rand (for session id suffix).
func randHex2() string {
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte{byte(time.Now().UnixNano() % 256)})
	}
	return hex.EncodeToString(b)
}

func loadConfig() (*config.Config, error) {
	if err := config.EnsureRootDir(); err != nil {
		return nil, err
	}
	return config.Load()
}

// ensureConfigWithWizard 确保 config.yaml 已存在；若缺失且未显式关闭向导，则运行首次启动向导。
// 返回值 ranWizard 表示本次是否执行了向导。
func ensureConfigWithWizard() (*config.Config, bool, error) {
	if err := config.EnsureRootDir(); err != nil {
		return nil, false, err
	}
	// 测试或特殊环境可通过 DELVE_SHELL_NO_WIZARD=1 关闭向导，回退到旧行为。
	if os.Getenv("DELVE_SHELL_NO_WIZARD") != "" {
		cfg, err := config.Load()
		return cfg, false, err
	}
	path := config.ConfigPath()
	if _, err := os.Stat(path); err == nil {
		cfg, err := config.Load()
		return cfg, false, err
	} else if !os.IsNotExist(err) {
		return nil, false, err
	}

	cfg, err := runFirstTimeWizard(path)
	if err != nil {
		return nil, false, err
	}
	if err := config.Write(cfg); err != nil {
		return nil, false, err
	}
	return cfg, true, nil
}

// runFirstTimeWizard 在终端中引导用户填写基础配置（base_url / api_key / model）。
// 仅在 config.yaml 不存在时调用。文案来自 i18n（当前为英文）。
func runFirstTimeWizard(configPath string) (*config.Config, error) {
	introLang := "en"
	fmt.Println(i18n.T(introLang, i18n.KeyWizardTitle))
	fmt.Println(i18n.Tf(introLang, i18n.KeyWizardConfigPath, configPath))
	fmt.Println(i18n.T(introLang, i18n.KeyWizardIntroDesc1))
	if s := i18n.T(introLang, i18n.KeyWizardIntroDesc2); s != "" {
		fmt.Println(s)
	}
	fmt.Println(i18n.T(introLang, i18n.KeyWizardIntroEnv))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	lang := "en"

	// Base URL
	fmt.Print(i18n.T(lang, i18n.KeyWizardBaseURLPrompt))
	baseURL, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	baseURL = strings.TrimSpace(baseURL)

	// API key (required)
	apiKey := ""
	for {
		fmt.Print(i18n.T(lang, i18n.KeyWizardAPIKeyPrompt))
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			fmt.Println(i18n.T(lang, i18n.KeyWizardAPIKeyRequired))
			continue
		}
		apiKey = line
		break
	}

	// Model
	fmt.Print(i18n.T(lang, i18n.KeyWizardModelPrompt))
	model, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = "gpt-4o-mini"
	}

	cfg := config.Default()
	cfg.LLM.BaseURL = baseURL
	cfg.LLM.APIKey = apiKey
	cfg.LLM.Model = model

	fmt.Println()
	fmt.Println(i18n.T(lang, i18n.KeyWizardDone))
	fmt.Println()
	return cfg, nil
}

// testLLMConnection 做一次最小化的 LLM 连通性测试，不执行任何命令或工具。
func testLLMConnection(ctx context.Context, cfg *config.Config) error {
	baseURL, apiKey, model := cfg.LLMResolved()
	if apiKey == "" {
		return agent.ErrLLMNotConfigured
	}
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return err
	}
	_, err = chatModel.Generate(ctx, []*schema.Message{
		schema.UserMessage("delve-shell config test: reply with single word OK."),
	})
	return err
}

