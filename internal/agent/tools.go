package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hil"
	"delve-shell/internal/history"
)

// ApprovalRequest is sent to HIL: pending command and response channel.
type ApprovalRequest struct {
	Command    string // command to run
	Reason     string // AI explanation (why, expected effect); may be empty
	RiskLevel  string // read_only | low | high; empty if not provided
	ResponseCh chan bool
}

// SensitiveChoice is the user's choice when a command may access sensitive path(s).
type SensitiveChoice int

const (
	SensitiveRefuse     SensitiveChoice = iota // 1: refuse, do not run
	SensitiveRunAndStore                       // 2: run, return result to AI, store in history
	SensitiveRunNoStore                        // 3: run, return result to AI, do not store in history
)

// SensitiveConfirmationRequest is sent to HIL when command may access sensitive file(s); user picks Refuse / RunAndStore / RunNoStore.
type SensitiveConfirmationRequest struct {
	Command    string
	ResponseCh chan SensitiveChoice
}

// ExecEvent is emitted after command execution for TUI to show HIL process and result.
type ExecEvent struct {
	Command   string
	Allowed   bool   // matched allowlist, no approval needed
	Result    string // stdout + stderr + exit_code for display
	Sensitive bool   // if true, result contains private data, not stored and LLM sees "done"
	Suggested bool   // if true, command was only suggested (suggest mode), not executed
}

// ExecuteCommandTool runs a command/script; blocks on requestApproval until user approves or rejects when not on allowlist.
// When command may access sensitive path(s), blocks on requestSensitiveConfirmation for user to choose: refuse / run+store / run+no store.
// When Mode is "suggest", no command is executed; all are recorded as suggested and OnExec is called with Suggested=true.
type ExecuteCommandTool struct {
	Mode       string // "suggest" or "run"; default "run"
	Allowlist  *hil.Allowlist
	SensitiveMatcher           *hil.SensitiveMatcher
	RequestApproval            func(command, reason, riskLevel string) bool
	RequestSensitiveConfirmation func(command string) SensitiveChoice
	Session                    *history.Session
	OnExec                     func(command string, allowed bool, result string, sensitive bool, suggested bool)
}

var _ tool.InvokableTool = (*ExecuteCommandTool)(nil)

func (t *ExecuteCommandTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "execute_command",
		Desc: "Execute a shell command or script in the user's environment. Prefer shell commands to accomplish tasks; use Python or other scripting only when shell is not sufficient. If the command is not on the allowlist, the tool waits for user approval before running. Results must not contain user secrets or passwords; if output may contain sensitive data, set result_contains_secrets to true so the result is shown only to the user and not returned to the model or stored in history.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "Full command or script to run (may include pipes, etc.)",
				Required: true,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "Brief explanation of why this command is run and what effect is expected. Shown to the user in the approval card.",
				Required: false,
			},
			"risk_level": {
				Type:     schema.String,
				Desc:     "Risk level: read_only (no side effects), low (e.g. read config), high (e.g. restart, delete). Used for approval UI only.",
				Required: false,
			},
			"result_contains_secrets": {
				Type:     schema.Boolean,
				Desc:     "Set to true if the command output may contain secrets, passwords, or other private data. When true, the result is shown only to the user; the model receives 'done' and the result is not stored in session history.",
				Required: false,
			},
		}),
	}, nil
}

func (t *ExecuteCommandTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input struct {
		Command               string `json:"command"`
		Reason                string `json:"reason"`
		RiskLevel             string `json:"risk_level"`
		ResultContainsSecrets bool   `json:"result_contains_secrets"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil || input.Command == "" {
		return "execute_command requires parameter 'command' (string)", nil
	}
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return "command cannot be empty", nil
	}
	reason := strings.TrimSpace(input.Reason)
	riskLevel := strings.TrimSpace(strings.ToLower(input.RiskLevel))
	if riskLevel != "" && riskLevel != "read_only" && riskLevel != "low" && riskLevel != "high" {
		riskLevel = "" // invalid value treated as not provided
	}
	sensitive := input.ResultContainsSecrets

	mode := strings.TrimSpace(strings.ToLower(t.Mode))
	if mode != "suggest" && mode != "run" {
		mode = "run"
	}
	if mode == "suggest" {
		if t.Session != nil {
			_ = t.Session.AppendSuggestedCommand(command, reason, riskLevel)
		}
		if t.OnExec != nil {
			t.OnExec(command, false, "(suggested, not executed)", false, true)
		}
		return "This command was only suggested and was not executed (suggest mode). The user can see it in the conversation and may copy or run it elsewhere. Continue with your reply or suggest next steps.", nil
	}

	approved := true
	allowed := false
	if t.Allowlist != nil {
		// any write redirection (>, >>, etc.) is never auto-allowed; must be approved by user
		allowed = !hil.ContainsWriteRedirection(command) &&
			t.Allowlist.AllowStrict(command)
	}
	if !allowed {
		approved = t.RequestApproval(command, reason, riskLevel)
		if t.Session != nil {
			_ = t.Session.AppendCommand(command, approved, reason, riskLevel)
		}
		if !approved {
			return "The user declined to run this command: " + command + ". Continue without running it; you may suggest an alternative or ask what they prefer.", nil
		}
	} else if t.Session != nil {
		_ = t.Session.AppendCommand(command, true, "", "")
	}

	// When command may access sensitive path(s), ask user: refuse / run+store / run+no store.
	storeResult := true
	if t.SensitiveMatcher != nil && t.SensitiveMatcher.MayAccessSensitivePath(command) && t.RequestSensitiveConfirmation != nil {
		choice := t.RequestSensitiveConfirmation(command)
		switch choice {
		case SensitiveRefuse:
			return "The user declined to run this command (may access sensitive file(s)): " + command + ". Continue without running it.", nil
		case SensitiveRunNoStore:
			storeResult = false // run, return result to AI, but do not store in history
		case SensitiveRunAndStore:
			// storeResult = true
		}
	} else if input.ResultContainsSecrets {
		storeResult = false
		sensitive = true
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	outStr := stdout.String()
	errStr := stderr.String()
	if storeResult && t.Session != nil {
		_ = t.Session.AppendCommandResult(command, outStr, errStr, exitCode)
	}
	resultForUI := outStr
	if errStr != "" {
		resultForUI += "\nstderr:\n" + errStr
	}
	resultForUI += "\nexit_code: " + strconv.Itoa(exitCode)
	if err != nil && exitCode == 0 {
		resultForUI += "\nerror: " + err.Error()
	}
	if t.OnExec != nil {
		t.OnExec(command, allowed, resultForUI, sensitive || !storeResult, false)
	}
	// When AI set result_contains_secrets we return "done"; when user chose RunNoStore we still return full result to AI.
	if sensitive {
		return "done", nil
	}
	msg := "stdout:\n" + outStr
	if errStr != "" {
		msg += "\nstderr:\n" + errStr
	}
	msg += "\nexit_code: " + strconv.Itoa(exitCode)
	if err != nil && exitCode == 0 {
		msg += "\nerror: " + err.Error()
	}
	return msg, nil
}

// ViewContextTool lets the AI fetch session context on demand (read-only).
type ViewContextTool struct {
	SessionPath string
	MaxEvents   int
}

var _ tool.InvokableTool = (*ViewContextTool)(nil)

func (t *ViewContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "view_context",
		Desc: "View current session history (user input, LLM replies, executed commands and results). Read-only. Use when you need to recall the conversation or command results.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"max_events": {
				Type: schema.Integer,
				Desc: "Max number of recent events to return; default 50; 0 means use default",
			},
		}),
	}, nil
}

func (t *ViewContextTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	max := t.MaxEvents
	if max <= 0 {
		max = 50
	}
	if argumentsInJSON != "" {
		var input struct {
			MaxEvents int `json:"max_events"`
		}
		_ = json.Unmarshal([]byte(argumentsInJSON), &input)
		if input.MaxEvents > 0 {
			max = input.MaxEvents
		}
	}
	if t.SessionPath == "" {
		return "no session history", nil
	}
	events, err := history.ReadRecent(t.SessionPath, max)
	if err != nil {
		return "read context failed: " + err.Error(), nil
	}
	var b strings.Builder
	for _, ev := range events {
		b.WriteString(ev.Time.Format("2006-01-02 15:04:05"))
		b.WriteString(" [")
		b.WriteString(ev.Type)
		b.WriteString("] ")
		b.WriteString(string(ev.Payload))
		b.WriteString("\n")
	}
	return b.String(), nil
}
