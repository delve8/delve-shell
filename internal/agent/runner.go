package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
)

// RunnerOptions for creating a Runner; LLM is read from Config (config.yaml, supports $VAR env expansion).
type RunnerOptions struct {
	Config                     *config.Config
	AllowlistAutoRun           *bool  // optional runtime override; when nil use Config.AllowlistAutoRunResolved()
	Allowlist                  *hil.Allowlist
	SensitiveMatcher           *hil.SensitiveMatcher
	Session                    *history.Session
	RulesText                  string
	ApprovalChan               chan<- *ApprovalRequest
	SensitiveConfirmationChan  chan<- *SensitiveConfirmationRequest
	ExecEventChan              chan<- ExecEvent
	ExecutorProvider           func() execenv.CommandExecutor // returns current executor (local or remote)
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

	requestApproval := func(cmd, summary, reason, riskLevel, skillName string) ApprovalResponse {
		ch := make(chan ApprovalResponse, 1)
		opts.ApprovalChan <- &ApprovalRequest{Command: cmd, Summary: summary, Reason: reason, RiskLevel: riskLevel, SkillName: strings.TrimSpace(skillName), ResponseCh: ch}
		return <-ch
	}
	requestSensitiveConfirmation := func(cmd string) SensitiveChoice {
		if opts.SensitiveConfirmationChan == nil {
			return SensitiveRunAndStore
		}
		ch := make(chan SensitiveChoice)
		opts.SensitiveConfirmationChan <- &SensitiveConfirmationRequest{Command: cmd, ResponseCh: ch}
		return <-ch
	}

	allowlistAutoRun := opts.Config.AllowlistAutoRunResolved()
	if opts.AllowlistAutoRun != nil {
		allowlistAutoRun = *opts.AllowlistAutoRun
	}
	execTool := &ExecuteCommandTool{
		AllowlistAutoRun:            allowlistAutoRun,
		Allowlist:                   opts.Allowlist,
		SensitiveMatcher:            opts.SensitiveMatcher,
		RequestApproval:             requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		Session:                     opts.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool, suggested bool) {
			if opts.ExecEventChan != nil {
				opts.ExecEventChan <- ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive, Suggested: suggested}
			}
		},
		ExecutorProvider: opts.ExecutorProvider,
	}
	viewTool := &ViewContextTool{
		SessionPath: "",
		MaxEvents:   50,
	}
	if opts.Session != nil {
		viewTool.SessionPath = opts.Session.Path()
	}
	listSkillsTool := &ListSkillsTool{}
	getSkillTool := &GetSkillTool{}
	runSkillTool := &RunSkillTool{
		RequestApproval:             requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		SensitiveMatcher:            opts.SensitiveMatcher,
		Session:                     opts.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool, suggested bool) {
			if opts.ExecEventChan != nil {
				opts.ExecEventChan <- ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive, Suggested: suggested}
			}
		},
		ExecutorProvider: opts.ExecutorProvider,
	}

	sysPrompt := opts.Config.LLM.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = defaultSystemPrompt
	}
	sysPrompt = config.ExpandEnv(sysPrompt)
	sysPrompt += "\n\n--- Auto-run ---\n" + autoRunParagraph(allowlistAutoRun)
	if opts.RulesText != "" {
		sysPrompt += "\n\n--- User rules (rules) ---\n" + opts.RulesText
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

const defaultSystemPrompt = `You are an ops assistant. You run commands in the user's environment via execute_command and can read session history via view_context. Installed skills (scripts under ~/.delve-shell/skills/) can be listed with list_skills and run with run_skill (same approval flow as execute_command).

## Execution strategy
- Prefer one execute_command call per user goal. Combine multiple steps into a single shell command (e.g. "cmd1 && cmd2 && cmd3" or pipelines) so the user approves once for the whole operation.
- Use multiple execute_command calls only when a later step must depend on the previous command's output to decide what to run next.
- Prefer shell; use Python or other tools only when shell is not sufficient.
- When you need to inspect multiple similar resources (e.g. several pods with errors), prefer a small number of batch commands (label selectors, namespaces, shell loops) instead of many single-resource commands.

## Approval and safety
- Commands not on the allowlist require explicit user approval in this tool. Do not "ask" in chat—the tool shows the pending command and waits for confirmation.
- For every execute_command call, always set reason (why this command and expected effect) and risk_level (read_only, low, or high) so the user sees a clear approval card.
- If command output may contain secrets or sensitive data, set result_contains_secrets to true: the result is shown only to the user, you receive "done", and it is not stored in history.

## Clarifications and confirmations
- When you need the user's decision, present explicit options and tell the user how to answer (for example: "Option 1: ..., Option 2: ...; reply with 1 or 2.").
- Avoid vague yes/no questions like "Do you need me to ...?". Instead, restate what you will do for each option so the meaning of the user's choice is unambiguous.
- Never ask in chat whether you should run a command or script; triggering execute_command is the only way to propose execution, and the approval card is the only place where the user approves or rejects it.

## Skills
- Skills live under ~/.delve-shell/skills/<name>/ with SKILL.md and scripts/ subdir. Use list_skills to discover all skills (name, description). Use get_skill(skill_name) to read one skill's full SKILL.md (usage, params, examples). Then call run_skill(skill_name, script_name, args=[...]) to run it (approval card like execute_command).
- Before run_skill: call get_skill(skill_name) so you have the full contract (which script, which args). Prefer run_skill when the user's goal matches an installed skill; otherwise use execute_command.

## Context
- Use view_context when you need to see recent session history (commands and results) to inform your next step.

## Loop control
- The agent has a limited number of internal steps per turn. Avoid calling tools repeatedly when they are failing in the same way.
- After a few unsuccessful or uninformative tool calls, stop retrying, explain the limitation, and summarize what you know so far.
- If more tool calls would only repeat earlier failures or add little value, give your best recommendation based on existing information instead of looping.`

func autoRunParagraph(allowlistAutoRun bool) string {
	if allowlistAutoRun {
		return `Auto-run: list only. Allowlisted commands run directly; others show an approval card (user can Run, Reject, or Copy). Commands with write redirection (>, >>) always show the card.`
	}
	return `Auto-run: none. Every command shows an approval card (Run, Copy, or Dismiss). No command runs without user choice. Prefer one combined command per task so the user approves once.`
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

// ErrNoAPIKey indicates LLM API Key was not set (deprecated, use ErrLLMNotConfigured).
var ErrNoAPIKey = errors.New("LLM API key not set")

// ErrLLMNotConfigured indicates LLM is not configured in config.yaml. Error() returns English text for logs; UI should show localized message via i18n.
var ErrLLMNotConfigured errLLMNotConfigured

type errLLMNotConfigured struct{}

func (errLLMNotConfigured) Error() string {
	return fmt.Sprintf("LLM not configured: set llm.api_key (and llm.base_url, llm.model) in %s or use /config", config.ConfigPath())
}
