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
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"
	"github.com/spf13/cobra"

	"delve-shell/internal/agent"
	"delve-shell/internal/app/runtime/executormgr"
	"delve-shell/internal/app/runtime/runnermgr"
	"delve-shell/internal/app/runtime/sessionmgr"
	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/modelinfo"
	"delve-shell/internal/rules"
	"delve-shell/internal/ui"
)

func runRun(cmd *cobra.Command, args []string) error {
	_ = args

	// stop is closed only when the whole CLI run exits (not when /sh temporarily leaves the TUI).
	stop := make(chan struct{})
	defer close(stop)

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
	sessions := sessionmgr.New(session)
	defer sessions.CloseAll()

	approvalChan := make(chan *agent.ApprovalRequest, 4)
	sensitiveConfirmationChan := make(chan *agent.SensitiveConfirmationRequest, 4)
	execEventChan := make(chan agent.ExecEvent, 8)
	configUpdatedChan := make(chan struct{}, 1)
	allowlistAutoRunChangeChan := make(chan bool, 1)

	cfg0, _ := loadConfig()
	var currentAllowlistAutoRun atomic.Bool
	currentAllowlistAutoRun.Store(true)
	if cfg0 != nil {
		currentAllowlistAutoRun.Store(cfg0.AllowlistAutoRunResolved())
	}

	executors := executormgr.New()

	// getExecutor returns the current executor for the Runner.
	getExecutor := func() execenv.CommandExecutor {
		return executors.Get()
	}

	runners := runnermgr.New(runnermgr.Options{
		RulesText: rulesText,
		LoadConfig: func() (*config.Config, error) {
			return loadConfig()
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
		ApprovalChan:     approvalChan,
		SensitiveConfirmationChan: sensitiveConfirmationChan,
		ExecEventChan:             execEventChan,
	})

	submitChan := make(chan string, 4)
	execDirectChan := make(chan string, 4)
	shellRequestedChan := make(chan []string, 1)
	cancelRequestChan := make(chan struct{}, 1)
	sessionSwitchChan := make(chan string, 1)
	remoteOnChan := make(chan string, 1)
	remoteOffChan := make(chan struct{}, 1)
	remoteAuthRespChan := make(chan ui.RemoteAuthResponse, 1)
	var savedMessages []string
	var currentP atomic.Pointer[tea.Program]
	// uiMsgChan serializes UI messages across goroutines so p.Send is called from one place.
	uiMsgChan := make(chan tea.Msg, 256)
	go func() {
		for {
			select {
			case <-stop:
				return
			case m := <-uiMsgChan:
				if m == nil {
					continue
				}
				if p := currentP.Load(); p != nil {
					p.Send(m)
				}
			}
		}
	}()
	sendToUI := func(msg tea.Msg) {
		if msg == nil {
			return
		}
		select {
		case uiMsgChan <- msg:
		default:
			// Best-effort: avoid blocking background goroutines on UI congestion.
		}
	}

	// updateRemoteRunCompletion fetches a one-time /run completion cache from the remote host.
	// It runs best-effort and never blocks the main connection flow.
	updateRemoteRunCompletion := func(exec execenv.CommandExecutor, remoteLabel string) {
		if currentP.Load() == nil || exec == nil || strings.TrimSpace(remoteLabel) == "" {
			return
		}
		go func() {
			select {
			case <-stop:
				return
			default:
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Use bash compgen for a reasonably complete command list.
			out, _, _, err := exec.Run(ctx, "bash -lc 'compgen -c'")
			if err != nil {
				return
			}
			seen := make(map[string]struct{}, 4096)
			cmds := make([]string, 0, 2048)
			for _, line := range strings.Split(out, "\n") {
				s := strings.TrimSpace(line)
				if s == "" {
					continue
				}
				// Keep it simple: name-like tokens only.
				if strings.ContainsAny(s, " \t/") {
					continue
				}
				if _, ok := seen[s]; ok {
					continue
				}
				seen[s] = struct{}{}
				cmds = append(cmds, s)
				// Cap to avoid excessive memory/CPU for huge environments.
				if len(cmds) >= 8000 {
					break
				}
			}
			sort.Strings(cmds)
			sendToUI(ui.RunCompletionCacheMsg{RemoteLabel: remoteLabel, Commands: cmds})
		}()
	}

	go func() {
		for {
			select {
			case <-stop:
				return
			case path := <-sessionSwitchChan:
				_, err := sessions.SwitchTo(path)
				if err != nil {
					sendToUI(ui.AgentReplyMsg{Err: err})
					continue
				}
				runners.Invalidate()
				sendToUI(ui.SessionSwitchedMsg{Path: path})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-configUpdatedChan:
				if cfg, err := loadConfig(); err == nil && cfg != nil {
					currentAllowlistAutoRun.Store(cfg.AllowlistAutoRunResolved())
				}
				runners.SetAllowlistAutoRun(currentAllowlistAutoRun.Load())
				sendToUI(ui.ConfigReloadedMsg{})
			case newAutoRun := <-allowlistAutoRunChangeChan:
				currentAllowlistAutoRun.Store(newAutoRun)
				runners.SetAllowlistAutoRun(newAutoRun)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case req := <-approvalChan:
				sendToUI(req)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case req := <-sensitiveConfirmationChan:
				sendToUI(req)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case ev := <-execEventChan:
				sendToUI(ui.CommandExecutedMsg{Command: ev.Command, Allowed: ev.Allowed, Result: ev.Result, Sensitive: ev.Sensitive, Suggested: ev.Suggested})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case cmd := <-execDirectChan:
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
			sendToUI(ui.CommandExecutedMsg{Command: cmd, Direct: true, Result: result})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case target := <-remoteOnChan:
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

			_ = hostOnly
			res := executors.Connect(target, label, identityFile)
			if res.AuthPrompt != nil {
				sendToUI(*res.AuthPrompt)
			}
			if !res.Connected {
				continue
			}
			updateRemoteRunCompletion(res.Executor, res.Label)
			sendToUI(ui.RemoteStatusMsg{Active: true, Label: res.Label})
			sendToUI(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", res.Label)})
			sendToUI(ui.RemoteConnectDoneMsg{Success: true, Label: res.Label})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-remoteOffChan:
				executors.SwitchToLocal()
			sendToUI(ui.RemoteStatusMsg{Active: false, Label: ""})
			sendToUI(ui.SystemNotifyMsg{Text: "Switched back to local executor."})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case resp := <-remoteAuthRespChan:
			if resp.Password == "" {
				continue
			}
			labelStr, err := executors.HandleRemoteAuthResponse(resp)
			if err != nil {
				sendToUI(ui.RemoteAuthPromptMsg{
					Target: resp.Target,
					Err:   fmt.Sprintf("Auth failed: %v", err),
				})
				continue
			}
			updateRemoteRunCompletion(getExecutor(), labelStr)
			sendToUI(ui.RemoteStatusMsg{Active: true, Label: labelStr})
			sendToUI(ui.SystemNotifyMsg{Text: fmt.Sprintf("Connected to remote: %s", labelStr)})
			sendToUI(ui.RemoteConnectDoneMsg{Success: true, Label: labelStr})
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case userMsg := <-submitChan:
			if userMsg == "/new" {
				newSession, err := sessions.NewSession(randHex2)
				if err != nil {
					sendToUI(ui.AgentReplyMsg{Err: err})
					continue
				}
				runners.Invalidate()
				sendToUI(ui.SessionSwitchedMsg{Path: newSession.Path()})
				continue
			}
			// Always record user input before calling the agent so audit history has the question
			// even if the LLM run fails (e.g. max steps exceeded, 5xx, or cancelled).
			if s := sessions.Current(); s != nil {
				_ = s.AppendUserInput(userMsg)
			}
			r, err := runners.Get(context.Background())
			if err != nil {
				sendToUI(ui.AgentReplyMsg{Err: err})
				continue
			}
			reqCtx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			var reply string
			var runErr error
			go func() {
				defer close(done)
				var historyMsgs []*schema.Message
				if s := sessions.Current(); s != nil {
					events, _ := history.ReadRecent(s.Path(), agent.MaxConversationEvents)
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
					sendToUI(ui.AgentReplyMsg{Err: runErr})
					continue
				}
				if s := sessions.Current(); s != nil {
					_ = s.AppendLLMResponse(map[string]string{"reply": reply})
				}
				sendToUI(ui.AgentReplyMsg{Reply: reply})
			case <-cancelRequestChan:
				cancel()
				<-done
				sendToUI(ui.AgentReplyMsg{Err: runErr})
			}
			}
		}
	}()

	getAllowlistAutoRun := func() bool { return currentAllowlistAutoRun.Load() }
	initialShowConfigLLM := needConfigLLM
	for {
		s := sessions.Current()
		sessionPath := ""
		if s != nil {
			sessionPath = s.Path()
		}
		model := ui.NewModel(submitChan, execDirectChan, shellRequestedChan, cancelRequestChan, configUpdatedChan, allowlistAutoRunChangeChan, sessionSwitchChan, remoteOnChan, remoteOffChan, remoteAuthRespChan, getAllowlistAutoRun, savedMessages, sessionPath, initialShowConfigLLM)
		initialShowConfigLLM = false
		// do not use WithMouse* so the terminal can use mouse for text selection; scroll with Up/Down/PgUp/PgDown
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
