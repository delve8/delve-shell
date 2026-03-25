package controller

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
	"delve-shell/internal/host/bus"
	"delve-shell/internal/modelinfo"
)

func (c *Controller) handleUserChat(userMsg string) {
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
		c.bus.PublishBlocking(bus.Event{
			Kind:  bus.KindLLMRunCompleted,
			Reply: reply,
			Err:   runErr,
		})
	}()
}

func (c *Controller) handleSubmitNewSession() {
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
}

func (c *Controller) handleSubmitSwitchSession(sessionID string) {
	if sessionID == "" {
		return
	}
	sessionPath := filepath.Join(config.HistoryDir(), sessionID+".jsonl")
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
