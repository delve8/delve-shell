package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/config"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
)

// RunnerOptions for creating a Runner; LLM is read from Config (config.yaml, supports $VAR env expansion).
type RunnerOptions struct {
	Config                     *config.Config
	Mode                       string // "suggest" or "run"; default "run"
	Allowlist                  *hil.Allowlist
	SensitiveMatcher           *hil.SensitiveMatcher
	Session                    *history.Session
	RulesText                  string
	ApprovalChan               chan<- *ApprovalRequest
	SensitiveConfirmationChan  chan<- *SensitiveConfirmationRequest
	ExecEventChan              chan<- ExecEvent
}

// Runner wraps the eino react agent; generates replies and runs commands via HIL approval.
type Runner struct {
	agent *react.Agent
}

// NewRunner creates a Runner; LLM from opts.Config.LLM, returns error guiding user to edit config.yaml if not set.
func NewRunner(ctx context.Context, opts RunnerOptions) (*Runner, error) {
	baseURL, apiKey, model := opts.Config.LLMResolved()
	if apiKey == "" {
		return nil, ErrLLMNotConfigured
	}

	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	if err != nil {
		return nil, err
	}

	requestApproval := func(cmd, reason, riskLevel string) bool {
		ch := make(chan bool)
		opts.ApprovalChan <- &ApprovalRequest{Command: cmd, Reason: reason, RiskLevel: riskLevel, ResponseCh: ch}
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

	mode := opts.Config.ModeResolved()
	if opts.Mode != "" {
		m := strings.TrimSpace(strings.ToLower(opts.Mode))
		if m == "suggest" || m == "run" {
			mode = m
		}
	}
	execTool := &ExecuteCommandTool{
		Mode:                        mode,
		Allowlist:                   opts.Allowlist,
		SensitiveMatcher:            opts.SensitiveMatcher,
		RequestApproval:              requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		Session:                     opts.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool, suggested bool) {
			if opts.ExecEventChan != nil {
				opts.ExecEventChan <- ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive, Suggested: suggested}
			}
		},
	}
	viewTool := &ViewContextTool{
		SessionPath: "",
		MaxEvents:   50,
	}
	if opts.Session != nil {
		viewTool.SessionPath = opts.Session.Path()
	}

	sysPrompt := opts.Config.LLM.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = defaultSystemPrompt
	}
	sysPrompt = config.ExpandEnv(sysPrompt)
	sysPrompt += "\n\n--- Current mode ---\n" + modeParagraph(mode)
	if opts.RulesText != "" {
		sysPrompt += "\n\n--- User rules (rules) ---\n" + opts.RulesText
	}

	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{execTool, viewTool},
		},
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

const defaultSystemPrompt = `You are an ops assistant. You run commands in the user's environment via execute_command and can read session history via view_context.

## Execution strategy
- Prefer one execute_command call per user goal. Combine multiple steps into a single shell command (e.g. "cmd1 && cmd2 && cmd3" or pipelines) so the user approves once for the whole operation.
- Use multiple execute_command calls only when a later step must depend on the previous command's output to decide what to run next.
- Prefer shell; use Python or other tools only when shell is not sufficient.

## Approval and safety
- Commands not on the allowlist require explicit user approval in this tool. Do not "ask" in chat—the tool shows the pending command and waits for confirmation.
- For every execute_command call, always set reason (why this command and expected effect) and risk_level (read_only, low, or high) so the user sees a clear approval card.
- If command output may contain secrets or sensitive data, set result_contains_secrets to true: the result is shown only to the user, you receive "done", and it is not stored in history.

## Context
- Use view_context when you need to see recent session history (commands and results) to inform your next step.`

func modeParagraph(mode string) string {
	switch mode {
	case "suggest":
		return `Current mode: suggest. execute_command does not run commands; it only records suggestions for the user. Still call it for each command you want to suggest so the user sees the list. Prefer one combined command per task. Tell the user these are suggestions only and they can copy or run them elsewhere.`
	default:
		return `Current mode: run. Allowlisted commands run directly; others require user approval. Commands with write redirection (>, >>) always require approval.`
	}
}

// Run generates a reply for one user message; blocks until user approves or rejects if agent calls a command requiring approval.
func (r *Runner) Run(ctx context.Context, userMessage string) (reply string, err error) {
	msg, err := r.agent.Generate(ctx, []*schema.Message{
		schema.UserMessage(userMessage),
	})
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
