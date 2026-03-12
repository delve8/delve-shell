package cli

import (
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
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/modelinfo"
	"delve-shell/internal/rules"
	"delve-shell/internal/ui"
)

func runRun(cmd *cobra.Command, args []string) error {
	_ = args

	if err := config.EnsureRootDir(); err != nil {
		return err
	}
	cfg, _ := loadConfig()
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

	// remoteCred stores in-memory SSH auth for a host (password or identity file).
	// Secrets are never written to disk; they live only for this process and are cleared on /remote off.
	type remoteCred struct {
		Kind     string // "password" or "identity"
		Username string
		Secret   string // password or identity file path
	}
	var remoteCredMu sync.Mutex
	remoteCreds := make(map[string]remoteCred) // key: host (without username)

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

			hostOnly := config.HostFromTarget(target)

			// First, try any cached credential for this host (password or identity).
			remoteCredMu.Lock()
			cred, hasCred := remoteCreds[hostOnly]
			remoteCredMu.Unlock()
			if hasCred {
				targetForSSH := target
				if cred.Username != "" {
					targetForSSH = cred.Username + "@" + hostOnly
				}
				var cachedExec execenv.CommandExecutor
				var err error
				switch cred.Kind {
				case "identity":
					cachedExec, _, err = execenv.NewSSHExecutor(targetForSSH, cred.Secret)
				default: // "password"
					cachedExec, _, err = execenv.NewSSHExecutorWithPassword(targetForSSH, "", cred.Secret)
				}
				if err == nil {
					if label == "" {
						label = hostOnly
					}
					executorMu.Lock()
					currentExecutor = cachedExec
					executorMu.Unlock()
					if currentP != nil {
						currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
						currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
						currentP.Send(ui.RemoteConnectDoneMsg{Success: true, Label: label})
					}
					continue
				}
				// Cached credential failed; drop it and fall back to config identityFile or interactive auth.
				remoteCredMu.Lock()
				delete(remoteCreds, hostOnly)
				remoteCredMu.Unlock()
			}

			// When an identity file is configured for this remote and there is no cached credential,
			// open Remote Auth dialog in "connecting with configured key" mode so the user sees the action,
			// then attempt the SSH connection immediately.
			if identityFile != "" {
				if currentP != nil {
					info := fmt.Sprintf("Using configured SSH key: %s", identityFile)
					currentP.Send(ui.RemoteAuthPromptMsg{
						Target:               target,
						Err:                  info,
						UseConfiguredIdentity: true,
					})
				}
				sshExec, _, err := execenv.NewSSHExecutor(target, identityFile)
				if err != nil {
					// On failure, fall back to interactive auth; keep Remote Auth dialog open and show error.
					if currentP != nil {
						msg := fmt.Sprintf("Remote connect failed for %s: %v", hostOnly, err)
						currentP.Send(ui.RemoteAuthPromptMsg{Target: target, Err: msg})
					}
					continue
				}
				if label == "" {
					label = hostOnly
				}
				executorMu.Lock()
				currentExecutor = sshExec
				executorMu.Unlock()
				if currentP != nil {
					currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
					currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
					currentP.Send(ui.RemoteConnectDoneMsg{Success: true, Label: label})
				}
				continue
			}

			// No identity file configured and no cached credential: try plain SSH and prompt on failure.
			sshExec, _, err := execenv.NewSSHExecutor(target, identityFile)
			if err != nil {
				// Authentication-related errors should trigger interactive auth prompt.
				if currentP != nil {
					msg := fmt.Sprintf("Remote connect failed for %s: %v", hostOnly, err)
					currentP.Send(ui.RemoteAuthPromptMsg{Target: target, Err: msg})
				}
				continue
			}

			if label == "" {
				label = hostOnly
			}
			executorMu.Lock()
			currentExecutor = sshExec
			executorMu.Unlock()
			if currentP != nil {
				currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
				currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
				currentP.Send(ui.RemoteConnectDoneMsg{Success: true, Label: label})
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
			// Clear any cached remote credentials when switching back to local.
			remoteCredMu.Lock()
			for k := range remoteCreds {
				delete(remoteCreds, k)
			}
			remoteCredMu.Unlock()
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
			hostOnly := config.HostFromTarget(resp.Target)
			if resp.Username != "" {
				targetForSSH = resp.Username + "@" + hostOnly
			}
			var sshExec execenv.CommandExecutor
			var err error
			switch resp.Kind {
			case "identity":
				sshExec, _, err = execenv.NewSSHExecutor(targetForSSH, resp.Password)
			default: // "password"
				sshExec, _, err = execenv.NewSSHExecutorWithPassword(targetForSSH, "", resp.Password)
			}
			if err != nil {
				// Best effort to clear password string on failure as well.
				resp.Password = ""
				if currentP != nil {
					currentP.Send(ui.RemoteAuthPromptMsg{
						Target: resp.Target,
						Err:   fmt.Sprintf("Auth failed: %v", err),
					})
				}
				continue
			}
			// Cache credential for this host so subsequent /remote on can reuse without prompting again.
			kind := resp.Kind
			if kind != "identity" {
				kind = "password"
			}
			remoteCredMu.Lock()
			remoteCreds[hostOnly] = remoteCred{
				Kind:     kind,
				Username: resp.Username,
				Secret:   resp.Password,
			}
			remoteCredMu.Unlock()
			// Best effort to clear password string after caching in-memory.
			resp.Password = ""
			executorMu.Lock()
			currentExecutor = sshExec
			executorMu.Unlock()
			if currentP != nil {
				label := config.HostFromTarget(targetForSSH)
				currentP.Send(ui.RemoteStatusMsg{Active: true, Label: label})
				currentP.Send(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", label)})
				currentP.Send(ui.RemoteConnectDoneMsg{Success: true, Label: label})
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
			// Always record user input before calling the agent so audit history has the question
			// even if the LLM run fails (e.g. max steps exceeded, 5xx, or cancelled).
			if session != nil {
				_ = session.AppendUserInput(userMsg)
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
				var historyMsgs []*schema.Message
				if session != nil {
					events, _ := history.ReadRecent(session.Path(), agent.MaxConversationEvents)
					historyMsgs = agent.BuildConversationMessages(events)
					if cfg, err := loadConfig(); err == nil && cfg != nil {
						maxMsg := cfg.MaxContextMessagesResolved()
						maxChars := cfg.MaxContextCharsResolved()
						if maxChars == 0 {
							baseURL, apiKey, modelName := cfg.LLMResolved()
							ctxTokens := modelinfo.FetchModelContextLength(baseURL, apiKey, modelName)
							if ctxTokens > 0 {
								// Use ~50% of context for history; ~4 chars per token
								maxChars = int(float64(ctxTokens) * 4 * 0.5)
							}
						}
						historyMsgs = agent.TrimConversationToContext(historyMsgs, maxMsg, maxChars)
					}
				}
				reply, runErr = r.Run(reqCtx, userMsg, historyMsgs)
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
	initialShowConfigLLM := needConfigLLM
	for {
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, allowlistAutoRunChangeChan, sessionSwitchChan, remoteOnChan, remoteOffChan, remoteAuthRespChan, getAllowlistAutoRun, savedMessages, session.Path(), initialShowConfigLLM)
		initialShowConfigLLM = false
		// do not use WithMouse* so the terminal can use mouse for text selection; scroll with Up/Down/PgUp/PgDown
		p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithReportFocus())
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
