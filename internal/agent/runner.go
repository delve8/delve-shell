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

	execTool := &ExecuteCommandTool{
		Allowlist:                    opts.Allowlist,
		SensitiveMatcher:             opts.SensitiveMatcher,
		RequestApproval:              requestApproval,
		RequestSensitiveConfirmation: requestSensitiveConfirmation,
		Session:                      opts.Session,
		OnExec: func(cmd string, allowed bool, result string, sensitive bool) {
			if opts.ExecEventChan != nil {
				opts.ExecEventChan <- ExecEvent{Command: cmd, Allowed: allowed, Result: result, Sensitive: sensitive}
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

const defaultSystemPrompt = `You are an ops assistant. Run commands in the user's environment via execute_command.

Prefer shell commands to accomplish tasks; only when shell is not sufficient, consider Python or other scripting tools in the environment.

Tool and script results must not contain user secrets, passwords, or other private data. If you must run something whose output may contain sensitive data, set execute_command's result_contains_secrets to true: the result will be shown only to the user, the model will receive "done", and the result will not be stored in session history.

Important: commands not on the allowlist must be explicitly approved by the user in this tool; do not rely on "asking" in chat—the tool will show the pending command and wait for confirmation. Use view_context when you need to see current session history.

When calling execute_command, always provide reason (brief explanation of why and expected effect) and risk_level (read_only, low, or high) so the user sees a clear approval card.`

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
