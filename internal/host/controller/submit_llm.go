package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/agent"
	"delve-shell/internal/agentctx"
	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/i18n"
	"delve-shell/internal/modelinfo"
	"delve-shell/internal/session"
	"delve-shell/internal/uivm"
)

// publishHistorySwitchDone sets the main transcript to a short switch line after a confirmed /history switch.
func (c *Controller) publishHistorySwitchDone(path string) {
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	lang := "en"
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	c.ui.ApplyHistorySwitchBanner(sessionID, lang)
}

func (c *Controller) handleHistoryPreviewOpen(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	if fields := strings.Fields(sessionID); len(fields) > 0 {
		sessionID = fields[0]
	}
	sessionPath := filepath.Join(config.HistoryDir(), sessionID+".jsonl")
	if _, err := os.Stat(sessionPath); err != nil {
		c.ui.AgentReply("", fmt.Errorf("history not found: %s", sessionID))
		return
	}
	lang := "en"
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	events, _ := history.ReadRecent(sessionPath, agent.MaxConversationEvents)
	vmLines := session.EventsToTranscriptLines(events)
	c.ui.ShowHistoryPreviewDialog(vmLines, sessionID, lang)
}

func (c *Controller) handleUserChat(e bus.Event) {
	sessionText := strings.TrimSpace(e.UserText)
	llmText := strings.TrimSpace(e.Submission.RawText)
	if llmText == "" {
		llmText = sessionText
	}
	if s := c.sessions.Current(); s != nil {
		_ = s.AppendUserInput(sessionText)
	}
	if c.llmRunning {
		c.ui.AgentReply("", fmt.Errorf("LLM request is already running; press Esc to cancel it first"))
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
	if n := strings.TrimSpace(e.Submission.SkillInvocationSkillName); n != "" {
		reqCtx = agentctx.WithSkillSlashTurn(reqCtx, n)
	}
	c.llmRunning = true
	c.llmCancel = cancel

	go func() {
		var historyMsgs []*schema.Message
		if s := c.sessions.Current(); s != nil {
			events, _ := history.ReadRecent(s.Path(), agent.MaxConversationEvents)
			historyMsgs = agent.BuildConversationMessages(events)
			if cfg, err := config.LoadEnsured(); err == nil && cfg != nil {
				maxMsg := cfg.LLM.MaxContextMessages
				if maxMsg <= 0 {
					maxMsg = 50
				}
				maxChars := cfg.LLM.MaxContextChars
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
		reply, runErr := r.Run(reqCtx, llmText, historyMsgs)
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
	c.publishSessionTranscript(newSession.Path())
}

func (c *Controller) handleSubmitSwitchSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	if fields := strings.Fields(sessionID); len(fields) > 0 {
		sessionID = fields[0]
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
	c.publishHistorySwitchDone(sessionPath)
}

// publishSessionTranscript loads recent events into the transcript, then appends a session banner (used for /new).
func (c *Controller) publishSessionTranscript(path string) {
	events, _ := history.ReadRecent(path, agent.MaxConversationEvents)
	lines := session.EventsToTranscriptLines(events)
	sessionID := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	lang := "en"
	if cfg, err := config.Load(); err == nil && cfg != nil && cfg.Language != "" {
		lang = cfg.Language
	}
	banner := i18n.Tf(lang, i18n.KeySessionSwitchedTo, sessionID)
	lines = append(lines, uivm.Line{Kind: uivm.LineSessionBanner, Text: banner})
	lines = append(lines, uivm.Line{Kind: uivm.LineBlank})
	c.ui.TranscriptReplace(lines)
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
