package hostcontroller

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/agent"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/history"
	"delve-shell/internal/hostbus"
	"delve-shell/internal/modelinfo"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/ui"
	"delve-shell/internal/uipresenter"
)

type Options struct {
	Stop <-chan struct{}

	Bus      *hostbus.Bus
	Inputs   hostbus.InputPorts
	CurrentP *atomic.Pointer[tea.Program]

	Sessions *sessionmgr.Manager
	Runners  *runnermgr.Manager

	Executors *executormgr.Manager
	GetExec   func() execenv.CommandExecutor

	CurrentAllowlistAutoRun *atomic.Bool

	SyncSessionPath func(path string)
}

// Controller is the single orchestration core for host-side flows.
type Controller struct {
	stop <-chan struct{}

	bus *hostbus.Bus

	ui *uipresenter.Presenter

	currentP *atomic.Pointer[tea.Program]

	sessions *sessionmgr.Manager
	runners  *runnermgr.Manager

	executors *executormgr.Manager
	getExec   func() execenv.CommandExecutor

	currentAllowlistAutoRun *atomic.Bool
	syncSessionPath         func(path string)

	fsm    *hostfsm.Machine
	fsmCtx hostfsm.Context

	llmRunning bool
	llmCancel  context.CancelFunc
}

func New(opts Options) *Controller {
	c := &Controller{
		stop: opts.Stop,

		bus: opts.Bus,
		ui:  uipresenter.New(uipresenter.BusSender{Bus: opts.Bus}),

		currentP: opts.CurrentP,

		sessions: opts.Sessions,
		runners:  opts.Runners,

		executors: opts.Executors,
		getExec:   opts.GetExec,

		currentAllowlistAutoRun: opts.CurrentAllowlistAutoRun,
		syncSessionPath:         opts.SyncSessionPath,

		fsm: hostfsm.NewMachine(hostfsm.StateIdle),
	}
	hostbus.BridgeInputs(opts.Stop, opts.Bus, opts.Inputs)
	hostbus.StartUIPump(opts.Stop, opts.Bus, opts.CurrentP)
	return c
}

func (c *Controller) Start() {
	go c.run()
}

func (c *Controller) run() {
	for {
		select {
		case <-c.stop:
			if c.llmRunning && c.llmCancel != nil {
				c.llmCancel()
			}
			return
		case e := <-c.bus.Events():
			c.handleEvent(e)
		}
	}
}

func (c *Controller) handleEvent(e hostbus.Event) {
	switch e.Kind {
	case hostbus.KindUserSubmitted:
		c.handleSubmit(e.UserText)
	case hostbus.KindConfigUpdated:
		c.handleConfigUpdated()
	case hostbus.KindCancelRequested:
		c.handleCancelRequest()
	case hostbus.KindExecDirectRequested:
		c.handleExecDirect(e.Command)
	case hostbus.KindRemoteOnRequested:
		c.handleRemoteOn(e.RemoteTarget)
	case hostbus.KindRemoteOffRequested:
		c.handleRemoteOff()
	case hostbus.KindRemoteAuthResponseSubmitted:
		c.handleRemoteAuthResp(e.RemoteAuthResponse)
	case hostbus.KindAgentUIEmitted:
		c.handleAgentUI(e.AgentUI)
	case hostbus.KindLLMRunCompleted:
		c.handleLLMRunCompleted(e.Reply, e.Err)
	}
}

func (c *Controller) handleCancelRequest() {
	if !c.llmRunning || c.llmCancel == nil {
		return
	}
	c.llmCancel()
}

func (c *Controller) handleConfigUpdated() {
	if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
		c.currentAllowlistAutoRun.Store(cfg.AllowlistAutoRunResolved())
	}
	c.runners.SetAllowlistAutoRun(c.currentAllowlistAutoRun.Load())
	c.ui.ConfigReloaded()
}

func (c *Controller) handleExecDirect(cmd string) {
	executor := c.getExec()
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
	c.ui.CommandExecutedDirect(cmd, result)
}

func (c *Controller) handleRemoteOn(target string) {
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
	res := c.executors.Connect(target, label, identityFile)
	c.ui.RemoteAuthPromptPtr(res.AuthPrompt)
	if !res.Connected {
		return
	}
	c.updateRemoteRunCompletion(res.Executor, res.Label)
	c.ui.RemoteStatus(true, res.Label)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", res.Label))
	c.ui.RemoteConnectDone(true, res.Label, "")
}

func (c *Controller) handleRemoteOff() {
	c.executors.SwitchToLocal()
	c.ui.RemoteStatus(false, "")
	c.ui.SystemNotify("Switched back to local executor.")
}

func (c *Controller) handleRemoteAuthResp(resp ui.RemoteAuthResponse) {
	if resp.Password == "" {
		return
	}
	labelStr, err := c.executors.HandleRemoteAuthResponse(resp)
	if err != nil {
		c.ui.RemoteAuthPrompt(ui.RemoteAuthPromptMsg{
			Target: resp.Target,
			Err:    fmt.Sprintf("Auth failed: %v", err),
		})
		return
	}
	c.updateRemoteRunCompletion(c.getExec(), labelStr)
	c.ui.RemoteStatus(true, labelStr)
	c.ui.SystemNotify(fmt.Sprintf("Connected to remote: %s", labelStr))
	c.ui.RemoteConnectDone(true, labelStr, "")
}

func (c *Controller) handleAgentUI(x any) {
	c.ui.DispatchAgentUI(x)
}

func (c *Controller) handleSubmit(userMsg string) {
	if userMsg == "/new" {
		newSession, err := c.sessions.NewSession()
		if err != nil {
			c.ui.AgentReply("", err)
			return
		}
		c.runners.Invalidate()
		if c.syncSessionPath != nil {
			c.syncSessionPath(newSession.Path())
		}
		c.ui.SessionSwitched()
		return
	}
	if strings.HasPrefix(userMsg, "/sessions ") {
		id := strings.TrimSpace(strings.TrimPrefix(userMsg, "/sessions "))
		if id == "" {
			return
		}
		sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
		_, err := c.sessions.SwitchTo(sessionPath)
		if err != nil {
			c.ui.AgentReply("", err)
			return
		}
		c.runners.Invalidate()
		if c.syncSessionPath != nil {
			c.syncSessionPath(sessionPath)
		}
		c.ui.SessionSwitched()
		return
	}
	if s := c.sessions.Current(); s != nil {
		_ = s.AppendUserInput(userMsg)
	}
	if c.llmRunning {
		c.ui.AgentReply("", fmt.Errorf("LLM request is already running; use /cancel first"))
		return
	}
	r, err := c.runners.Get(context.Background())
	if err != nil {
		c.ui.AgentReply("", err)
		return
	}
	if !c.fsm.Apply(&c.fsmCtx, hostfsm.EvtLLMRunStart) {
		c.ui.AgentReply("", fmt.Errorf("host FSM: cannot start LLM from state %q", c.fsm.State()))
		return
	}
	reqCtx, cancel := context.WithCancel(context.Background())
	c.llmRunning = true
	c.llmCancel = cancel

	go func() {
		var historyMsgs []*schema.Message
		if s := c.sessions.Current(); s != nil {
			events, _ := history.ReadRecent(s.Path(), agent.MaxConversationEvents)
			historyMsgs = agent.BuildConversationMessages(events)
			if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
				maxMsg := cfg.MaxContextMessagesResolved()
				maxChars := cfg.MaxContextCharsResolved()
				if maxChars == 0 {
					// Best-effort context budget: model context length * 4 chars/token * 0.5 safety.
					baseURL, apiKey, modelName := cfg.LLMResolved()
					ctxTokens := modelinfo.FetchModelContextLength(baseURL, apiKey, modelName)
					if ctxTokens > 0 {
						maxChars = int(float64(ctxTokens) * 4 * 0.5)
					}
				}
				historyMsgs = agent.TrimConversationToContext(historyMsgs, maxMsg, maxChars)
			}
		}
		reply, runErr := r.Run(reqCtx, userMsg, historyMsgs)
		c.bus.PublishBlocking(hostbus.Event{
			Kind:  hostbus.KindLLMRunCompleted,
			Reply: reply,
			Err:   runErr,
		})
	}()
}

func (c *Controller) handleLLMRunCompleted(reply string, runErr error) {
	if c.llmCancel != nil {
		c.llmCancel()
	}
	c.llmCancel = nil
	c.llmRunning = false
	_ = c.fsm.Apply(&c.fsmCtx, hostfsm.EvtLLMRunEnd)

	if runErr != nil {
		if strings.Contains(runErr.Error(), "404") {
			runErr = errors.Join(runErr, fmt.Errorf("%s", "Hint: For DashScope, ensure LLM_BASE_URL and API Key region match (Beijing vs International). See README for curl test."))
		}
		c.ui.AgentReply("", runErr)
		return
	}
	if s := c.sessions.Current(); s != nil {
		_ = s.AppendLLMResponse(map[string]string{"reply": reply})
	}
	c.ui.AgentReply(reply, nil)
}

func (c *Controller) updateRemoteRunCompletion(exec execenv.CommandExecutor, remoteLabel string) {
	if c.currentP.Load() == nil || exec == nil || strings.TrimSpace(remoteLabel) == "" {
		return
	}
	go func() {
		select {
		case <-c.stop:
			return
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
			if strings.ContainsAny(s, " \t/") {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			cmds = append(cmds, s)
			if len(cmds) >= 8000 {
				break
			}
		}
		sort.Strings(cmds)
		c.ui.RunCompletionCache(remoteLabel, cmds)
	}()
}

func (c *Controller) SyncCurrentSessionPath() {
	if c.syncSessionPath == nil {
		return
	}
	if s := c.sessions.Current(); s != nil {
		c.syncSessionPath(s.Path())
	}
}
