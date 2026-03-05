package cli

import (
	"bufio"
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
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
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
	sessionID := time.Now().Format("20060102-150405") + "-" + uuid.New().String()[:8]
	session, err := history.NewSession(sessionID)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	approvalChan := make(chan *agent.ApprovalRequest, 4)
	sensitiveConfirmationChan := make(chan *agent.SensitiveConfirmationRequest, 4)
	execEventChan := make(chan agent.ExecEvent, 8)
	configUpdatedChan := make(chan struct{}, 1)
	modeChangeChan := make(chan string, 1)

	cfg0, _ := loadConfig()
	currentMode := "run"
	if cfg0 != nil {
		currentMode = cfg0.ModeResolved()
	}

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
			Mode:                      currentMode,
			Allowlist:                 allowlist,
			SensitiveMatcher:          sensitiveMatcher,
			Session:                   session,
			RulesText:                 rulesText,
			ApprovalChan:              approvalChan,
			SensitiveConfirmationChan: sensitiveConfirmationChan,
			ExecEventChan:             execEventChan,
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
		for {
			select {
			case <-configUpdatedChan:
				if cfg, err := loadConfig(); err == nil && cfg != nil {
					currentMode = cfg.ModeResolved()
				}
				runnerMu.Lock()
				runner = nil
				runnerMu.Unlock()
				if currentP != nil {
					currentP.Send(ui.ConfigReloadedMsg{})
				}
			case newMode := <-modeChangeChan:
				currentMode = newMode
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

	getMode := func() string { return currentMode }
	for {
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, modeChangeChan, getMode, savedMessages)
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

// runFirstTimeWizard 在终端中引导用户填写基础配置（language / base_url / api_key / model）。
// 仅在 config.yaml 不存在时调用。文案来自 i18n，选择语言后后续提示使用该语言。
func runFirstTimeWizard(configPath string) (*config.Config, error) {
	introLang := "en" // 选择语言前的说明用英文（可改为根据 locale 推断）
	fmt.Println(i18n.T(introLang, i18n.KeyWizardTitle))
	fmt.Println(i18n.Tf(introLang, i18n.KeyWizardConfigPath, configPath))
	fmt.Println(i18n.T(introLang, i18n.KeyWizardIntroDesc1))
	if s := i18n.T(introLang, i18n.KeyWizardIntroDesc2); s != "" {
		fmt.Println(s)
	}
	fmt.Println(i18n.T(introLang, i18n.KeyWizardIntroEnv))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Language
	lang := ""
	for {
		fmt.Print(i18n.T(introLang, i18n.KeyWizardLangPrompt))
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			lang = "en"
			break
		}
		if line == "en" || line == "zh" {
			lang = line
			break
		}
		fmt.Println(i18n.T(introLang, i18n.KeyWizardLangInvalid))
	}

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
	cfg.Language = lang
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

