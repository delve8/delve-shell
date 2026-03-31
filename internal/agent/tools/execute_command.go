package tools

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hil"
	"delve-shell/internal/hil/types"
	"delve-shell/internal/history"
	"delve-shell/internal/remote/execenv"
)

// ExecuteCommandTool runs a command/script; blocks on requestApproval until user chooses Run or Reject when the command is not allowlisted (or has write redirection).
// When command may access sensitive path(s), blocks on requestSensitiveConfirmation for user to choose: refuse / run+store / run+no store.
// Allowlist: when non-nil, matching commands run without the approval card; an empty allowlist matches nothing.
type ExecuteCommandTool struct {
	Allowlist                    *hil.Allowlist
	SensitiveMatcher             *hil.SensitiveMatcher
	RequestApproval              func(command, summary, reason, riskLevel, skillName string) hiltypes.ApprovalResponse
	RequestSensitiveConfirmation func(command string) hiltypes.SensitiveChoice
	Session                      *history.Session
	OnExec                       func(command string, allowed bool, result string, sensitive bool, suggested bool)

	// ExecutorProvider returns the current executor (local or remote). When nil, a local executor is used.
	ExecutorProvider func() execenv.CommandExecutor

	// OfflineMode when true: skip allowlist and executor; use RequestOfflinePaste instead of approval+run.
	OfflineMode func() bool
	// RequestOfflinePaste blocks until the user pastes output or cancels (offline mode only).
	RequestOfflinePaste func(command, reason, riskLevel string) hiltypes.OfflinePasteResponse
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

	if t.OfflineMode != nil && t.OfflineMode() {
		return t.invokableRunOffline(ctx, command, reason, riskLevel, sensitive)
	}

	allowed := false
	if t.Allowlist != nil {
		allowed = !hil.ContainsWriteRedirection(command) &&
			t.Allowlist.AllowStrict(command)
	}
	if !allowed {
		resp := t.RequestApproval(command, "", reason, riskLevel, "")
		if t.Session != nil {
			_ = t.Session.AppendCommand(command, resp.Approved, reason, riskLevel, "", "")
		}
		if resp.CopyRequested {
			if t.Session != nil {
				_ = t.Session.AppendSuggestedCommand(command, reason, riskLevel, "", "")
			}
			return "The user copied the command and did not run it. Continue with your reply or suggest next steps.", nil
		}
		if !resp.Approved {
			return "The user declined to run this command: " + command + ". Continue without running it; you may suggest an alternative or ask what they prefer.", nil
		}
	} else if t.Session != nil {
		_ = t.Session.AppendCommand(command, true, "", "", "", "")
	}

	// When command may access sensitive path(s), ask user: refuse / run+store / run+no store.
	storeResult := true
	if t.SensitiveMatcher != nil && t.SensitiveMatcher.MayAccessSensitivePath(command) && t.RequestSensitiveConfirmation != nil {
		choice := t.RequestSensitiveConfirmation(command)
		switch choice {
		case hiltypes.SensitiveRefuse:
			return "The user declined to run this command (may access sensitive file(s)): " + command + ". Continue without running it.", nil
		case hiltypes.SensitiveRunNoStore:
			storeResult = false // run, return result to AI, but do not store in history
		case hiltypes.SensitiveRunAndStore:
			// storeResult = true
		}
	} else if input.ResultContainsSecrets {
		storeResult = false
		sensitive = true
	}

	executor := execenv.CommandExecutor(execenv.LocalExecutor{})
	if t.ExecutorProvider != nil {
		if e := t.ExecutorProvider(); e != nil {
			executor = e
		}
	}
	outStr, errStr, exitCode, err := executor.Run(ctx, command)
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

const manualPasteNoteForUI = "Manual paste — may be edited or mistaken."

func (t *ExecuteCommandTool) invokableRunOffline(ctx context.Context, command, reason, riskLevel string, resultContainsSecrets bool) (string, error) {
	_ = ctx
	sensitive := resultContainsSecrets
	storeResult := true
	if t.SensitiveMatcher != nil && t.SensitiveMatcher.MayAccessSensitivePath(command) && t.RequestSensitiveConfirmation != nil {
		choice := t.RequestSensitiveConfirmation(command)
		switch choice {
		case hiltypes.SensitiveRefuse:
			return "The user declined (sensitive path): " + command + ". Continue without this command.", nil
		case hiltypes.SensitiveRunNoStore:
			storeResult = false
			sensitive = true
		case hiltypes.SensitiveRunAndStore:
			// storeResult = true
		}
	} else if resultContainsSecrets {
		storeResult = false
		sensitive = true
	}
	if t.RequestOfflinePaste == nil {
		return "offline paste UI is not available", nil
	}
	paste := t.RequestOfflinePaste(command, reason, riskLevel)
	if paste.Cancelled {
		return "The user cancelled pasting output for: " + command + ". Continue without this result.", nil
	}
	pasted := strings.TrimSpace(paste.Text)
	if t.Session != nil {
		_ = t.Session.AppendOfflineCommandProposal(command, reason, riskLevel)
		if storeResult {
			_ = t.Session.AppendOfflinePasteResult(command, pasted)
		}
	}
	resultForUI := pasted
	if resultForUI != "" {
		resultForUI += "\n\n" + manualPasteNoteForUI
	} else {
		resultForUI = manualPasteNoteForUI
	}
	if t.OnExec != nil {
		t.OnExec(command, false, resultForUI, sensitive || !storeResult, false)
	}
	if sensitive {
		return "done", nil
	}
	if pasted == "" {
		return "The user submitted empty pasted output for: " + command + ".", nil
	}
	return "stdout:\n" + pasted, nil
}
