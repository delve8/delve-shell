package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	agenttools "delve-shell/internal/agent/tools"
	"delve-shell/internal/config"
	"delve-shell/internal/consts"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/hiltypes"
)

// RunnerHILInput is allowlist and sensitive matching for tools and approval flow.
type RunnerHILInput struct {
	Allowlist        *hil.Allowlist
	SensitiveMatcher *hil.SensitiveMatcher
}

// RunnerSessionInput is the active history session and injected rules text for the system prompt.
type RunnerSessionInput struct {
	Session   *history.Session
	RulesText string
}

// RunnerUILoopInput connects the agent to host-side UI and command execution.
type RunnerUILoopInput struct {
	// UIEvents sends *ApprovalRequest, *SensitiveConfirmationRequest, or ExecEvent to the host (e.g. TUI).
	// If nil: sensitive confirmation defaults to SensitiveRunAndStore; exec notifications are dropped; approvals are rejected without UI.
	UIEvents         chan<- any
	ExecutorProvider func() execenv.CommandExecutor // returns current executor (local or remote)
}

// RunnerOptions for creating a Runner; LLM is read from Config (config.yaml, supports $VAR env expansion).
type RunnerOptions struct {
	Config  *config.Config
	HIL     RunnerHILInput
	Session RunnerSessionInput
	UILoop  RunnerUILoopInput
}

// Runner wraps the eino react agent; generates replies and runs commands via HIL approval.
type Runner struct {
	agent *react.Agent
}

// NewRunner creates a Runner; LLM from opts.Config.LLM, returns error guiding user to edit config.yaml if not set.
func NewRunner(ctx context.Context, opts RunnerOptions) (*Runner, error) {
	baseURL, apiKey, model := opts.Config.LLMResolved()
	// apiKey may be empty for local deployments (e.g. Ollama) that do not require auth.

	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return nil, err
	}

	uiEvents := opts.UILoop.UIEvents
	requestApproval := func(cmd, summary, reason, riskLevel, skillName string) hiltypes.ApprovalResponse {
		if uiEvents == nil {
			return hiltypes.ApprovalResponse{}
		}
		ch := make(chan hiltypes.ApprovalResponse, 1)
		uiEvents <- &hiltypes.ApprovalRequest{Command: cmd, Summary: summary, Reason: reason, RiskLevel: riskLevel, SkillName: strings.TrimSpace(skillName), ResponseCh: ch}
		return <-ch
	}
	requestSensitiveConfirmation := func(cmd string) hiltypes.SensitiveChoice {
		if uiEvents == nil {
			return hiltypes.SensitiveRunAndStore
		}
		ch := make(chan hiltypes.SensitiveChoice)
		uiEvents <- &hiltypes.SensitiveConfirmationRequest{Command: cmd, ResponseCh: ch}
		return <-ch
	}

	execTool := &agenttools.ExecuteCommandTool{
		Allowlist:                    opts.HIL.Allowlist,
		SensitiveMatcher:             opts.HIL.SensitiveMatcher,
		RequestApproval:              requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		Session:                      opts.Session.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool, suggested bool) {
			if uiEvents != nil {
				uiEvents <- hiltypes.ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive, Suggested: suggested}
			}
		},
		ExecutorProvider: opts.UILoop.ExecutorProvider,
	}
	viewTool := &agenttools.ViewContextTool{
		SessionPath: "",
		MaxEvents:   50,
	}
	if opts.Session.Session != nil {
		viewTool.SessionPath = opts.Session.Session.Path()
	}
	listSkillsTool := &agenttools.ListSkillsTool{}
	getSkillTool := &agenttools.GetSkillTool{}
	runSkillTool := &agenttools.RunSkillTool{
		RequestApproval:              requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		SensitiveMatcher:             opts.HIL.SensitiveMatcher,
		Session:                      opts.Session.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool, suggested bool) {
			if uiEvents != nil {
				uiEvents <- hiltypes.ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive, Suggested: suggested}
			}
		},
		ExecutorProvider: opts.UILoop.ExecutorProvider,
	}

	sysPrompt := opts.Config.LLM.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = consts.DefaultSystemPrompt
	}
	sysPrompt = config.ExpandEnv(sysPrompt)
	sysPrompt += "\n\n--- Allowlist execution ---\n" + allowlistExecutionParagraph()
	if opts.Session.RulesText != "" {
		sysPrompt += "\n\n--- User rules (rules) ---\n" + opts.Session.RulesText
	}

	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{execTool, viewTool, listSkillsTool, getSkillTool, runSkillTool},
		},
		// Limit total ReAct steps per turn to avoid infinite loops; default is node count + 2.
		// 50 allows multiple tool calls (e.g. inspecting several pods) plus retries while still failing fast on loops.
		MaxStep: 50,
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			out := make([]*schema.Message, 0, len(input)+1)
			out = append(out, schema.SystemMessage(sysPrompt))
			out = append(out, input...)
			return out
		},
	})
	if err != nil {
		return nil, err
	}

	return &Runner{agent: reactAgent}, nil
}

func allowlistExecutionParagraph() string {
	return `Commands that match the allowlist (and have no shell write redirection like > or >>) run without an approval card. All other commands show an approval card: Run, Copy (clipboard, no execution), or Dismiss. An empty allowlist means nothing matches, so every command shows the card. Prefer one combined command per task when the user must approve.`
}

// MaxConversationEvents is the max number of session events to use when building conversation history (user_input + llm_response only).
const MaxConversationEvents = 200

// BuildConversationMessages converts session events to chat messages (user/assistant only) for context. Used so the model receives prior turns without calling view_context.
func BuildConversationMessages(events []history.Event) []*schema.Message {
	var out []*schema.Message
	for _, ev := range events {
		switch ev.Type {
		case "user_input":
			var p struct {
				Text string `json:"text"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil && p.Text != "" {
				out = append(out, schema.UserMessage(p.Text))
			}
		case "llm_response":
			var p struct {
				Reply string `json:"reply"`
			}
			if json.Unmarshal(ev.Payload, &p) == nil {
				out = append(out, schema.AssistantMessage(p.Reply, nil))
			}
		}
	}
	return out
}

// messageContentLength returns the character length of a message's text content for context budgeting.
func messageContentLength(m *schema.Message) int {
	if m == nil {
		return 0
	}
	return len(m.Content)
}

// TrimConversationToContext keeps the most recent messages that fit within maxMessages and (optionally) maxChars.
// maxMessages: 0 = do not limit by count; otherwise keep at most the last maxMessages.
// maxChars: 0 = do not limit by length; otherwise drop oldest messages until total content length <= maxChars.
func TrimConversationToContext(msgs []*schema.Message, maxMessages, maxChars int) []*schema.Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := msgs
	if maxMessages > 0 && len(out) > maxMessages {
		out = out[len(out)-maxMessages:]
	}
	if maxChars > 0 {
		total := 0
		for i := len(out) - 1; i >= 0; i-- {
			total += messageContentLength(out[i])
			if total > maxChars {
				out = out[i+1:]
				break
			}
		}
	}
	return out
}

// Run generates a reply for one user message; blocks until user approves or rejects if agent calls a command requiring approval.
// conversationHistory is optional: when non-nil and non-empty, it is prepended so the model sees prior user/assistant turns (e.g. from session). The current userMessage is always appended.
func (r *Runner) Run(ctx context.Context, userMessage string, conversationHistory []*schema.Message) (reply string, err error) {
	var input []*schema.Message
	if len(conversationHistory) > 0 {
		input = make([]*schema.Message, 0, len(conversationHistory)+1)
		input = append(input, conversationHistory...)
		input = append(input, schema.UserMessage(userMessage))
	} else {
		input = []*schema.Message{schema.UserMessage(userMessage)}
	}
	msg, err := r.agent.Generate(ctx, input)
	if err != nil {
		return "", err
	}
	if msg == nil {
		return "", nil
	}
	return strings.TrimSpace(msg.Content), nil
}

// ErrLLMNotConfigured indicates LLM is not configured in config.yaml. Error() returns English text for logs; UI should show localized message via i18n.
var ErrLLMNotConfigured errLLMNotConfigured

type errLLMNotConfigured struct{}

func (errLLMNotConfigured) Error() string {
	return fmt.Sprintf("LLM not configured: set llm.api_key (and llm.base_url, llm.model) in %s or use /config", config.ConfigPath())
}
