package hostloop

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/agent"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/modelinfo"
	"delve-shell/internal/ui"
)

// RunSubmitLoop handles user submit messages: /new, LLM runs (FSM: idle ↔ llm_running), cancellation.
func RunSubmitLoop(stop <-chan struct{}, d *Deps, submitChan <-chan string, cancelRequestChan <-chan struct{}, fsm *hostfsm.Machine) {
	var fsmCtx hostfsm.Context
	for {
		select {
		case <-stop:
			return
		case userMsg := <-submitChan:
			handleSubmit(&fsmCtx, d, cancelRequestChan, fsm, userMsg)
		}
	}
}

func handleSubmit(fsmCtx *hostfsm.Context, d *Deps, cancelRequestChan <-chan struct{}, fsm *hostfsm.Machine, userMsg string) {
	if userMsg == "/new" {
		newSession, err := d.Sessions.NewSession()
		if err != nil {
			d.Send(ui.AgentReplyMsg{Err: err})
			return
		}
		d.Runners.Invalidate()
		if d.SyncSessionPath != nil {
			d.SyncSessionPath(newSession.Path())
		}
		d.Send(ui.SessionSwitchedMsg{})
		return
	}
	if strings.HasPrefix(userMsg, "/sessions ") {
		id := strings.TrimSpace(strings.TrimPrefix(userMsg, "/sessions "))
		if id == "" {
			return
		}
		sessionPath := filepath.Join(config.HistoryDir(), id+".jsonl")
		_, err := d.Sessions.SwitchTo(sessionPath)
		if err != nil {
			d.Send(ui.AgentReplyMsg{Err: err})
			return
		}
		d.Runners.Invalidate()
		if d.SyncSessionPath != nil {
			d.SyncSessionPath(sessionPath)
		}
		d.Send(ui.SessionSwitchedMsg{})
		return
	}
	if s := d.Sessions.Current(); s != nil {
		_ = s.AppendUserInput(userMsg)
	}
	r, err := d.Runners.Get(context.Background())
	if err != nil {
		d.Send(ui.AgentReplyMsg{Err: err})
		return
	}
	if !fsm.Apply(fsmCtx, hostfsm.EvtLLMRunStart) {
		d.Send(ui.AgentReplyMsg{Err: fmt.Errorf("host FSM: cannot start LLM from state %q", fsm.State())})
		return
	}
	defer func() { _ = fsm.Apply(fsmCtx, hostfsm.EvtLLMRunEnd) }()

	reqCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	var reply string
	var runErr error
	go func() {
		defer close(done)
		var historyMsgs []*schema.Message
		if s := d.Sessions.Current(); s != nil {
			events, _ := history.ReadRecent(s.Path(), agent.MaxConversationEvents)
			historyMsgs = agent.BuildConversationMessages(events)
			if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
				maxMsg := cfg.MaxContextMessagesResolved()
				maxChars := cfg.MaxContextCharsResolved()
				if maxChars == 0 {
					// Best-effort context budget: FetchModelContextLength uses HTTP (cached); 0 => fall back to maxMsg-only trim.
					baseURL, apiKey, modelName := cfg.LLMResolved()
					ctxTokens := modelinfo.FetchModelContextLength(baseURL, apiKey, modelName)
					if ctxTokens > 0 {
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
			d.Send(ui.AgentReplyMsg{Err: runErr})
			return
		}
		if s := d.Sessions.Current(); s != nil {
			_ = s.AppendLLMResponse(map[string]string{"reply": reply})
		}
		d.Send(ui.AgentReplyMsg{Reply: reply})
	case <-cancelRequestChan:
		cancel()
		<-done
		d.Send(ui.AgentReplyMsg{Err: runErr})
	}
}
