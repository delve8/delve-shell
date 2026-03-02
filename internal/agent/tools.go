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

// ApprovalRequest 用于 HIL：待审批命令与回写通道
type ApprovalRequest struct {
	Command    string
	ResponseCh chan bool
}

// ExecEvent 命令执行后发出，供 TUI 展示 HIL 过程与结果
type ExecEvent struct {
	Command    string
	Whitelisted bool
	Result     string // stdout + stderr + exit_code，供界面展示
	Sensitive  bool   // 为 true 时结果含隐私数据，未写入历史且返回给 LLM 的为 "done"
}

// ExecuteCommandTool 执行命令/脚本；未命中白名单时通过 requestApproval 阻塞直至用户批准或拒绝
type ExecuteCommandTool struct {
	Whitelist       *hil.Whitelist
	RequestApproval func(command string) bool
	Session         *history.Session
	OnExec          func(command string, whitelisted bool, result string, sensitive bool)
}

var _ tool.InvokableTool = (*ExecuteCommandTool)(nil)

func (t *ExecuteCommandTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "execute_command",
		Desc: "Execute a shell command or script in the user's environment. Prefer shell commands to accomplish tasks; use Python or other scripting only when shell is not sufficient. If the command is not on the whitelist, the tool waits for user approval before running. Results must not contain user secrets or passwords; if output may contain sensitive data, set result_contains_secrets to true so the result is shown only to the user and not returned to the model or stored in history.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "Full command or script to run (may include pipes, etc.)",
				Required: true,
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
		ResultContainsSecrets bool   `json:"result_contains_secrets"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil || input.Command == "" {
		return "execute_command requires parameter 'command' (string)", nil
	}
	command := strings.TrimSpace(input.Command)
	if command == "" {
		return "command cannot be empty", nil
	}
	sensitive := input.ResultContainsSecrets

	approved := true
	whitelisted := false
	if t.Whitelist != nil {
		whitelisted = t.Whitelist.Allow(command) || t.Whitelist.AllowPipeline(command)
	}
	if !whitelisted {
		approved = t.RequestApproval(command)
		if t.Session != nil {
			_ = t.Session.AppendCommand(command, approved)
		}
		if !approved {
			return "User declined to run the command", nil
		}
	} else if t.Session != nil {
		_ = t.Session.AppendCommand(command, true)
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
	if !sensitive && t.Session != nil {
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
		t.OnExec(command, whitelisted, resultForUI, sensitive)
	}
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

// ViewContextTool 供 AI 按需拉取会话上下文（只读）
type ViewContextTool struct {
	SessionPath string
	MaxEvents   int
}

var _ tool.InvokableTool = (*ViewContextTool)(nil)

func (t *ViewContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "view_context",
		Desc: "查看当前会话的历史上下文（用户输入、LLM 回复、已执行命令及结果），用于需要回顾对话或命令结果时调用。只读。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"max_events": {
				Type: schema.Integer,
				Desc: "最多返回最近多少条事件，默认 50；0 表示使用默认",
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
