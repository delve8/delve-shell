package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hil"
	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/history"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/execcancel"
)

// ExecuteCommandTool runs a command/script; blocks on requestApproval until user chooses Run or Reject when the command is not allowlisted (or has write redirection).
// When command may access sensitive path(s), blocks on requestSensitiveConfirmation for user to choose: refuse / run+store / run+no store.
// Allowlist: when non-nil, matching commands run without the approval card; an empty allowlist matches nothing.
type ExecuteCommandTool struct {
	Allowlist                    *hil.Allowlist
	SensitiveMatcher             *hil.SensitiveMatcher
	RequestApproval              func(command, summary, reason, riskLevel, skillName string, autoApproveHL []hiltypes.AutoApproveHighlightSpan) hiltypes.ApprovalResponse
	RequestSensitiveConfirmation func(command string) hiltypes.SensitiveChoice
	Session                      *history.Session
	OnExec                       func(command string, allowed bool, result string, sensitive bool, suggested bool, offlineManual bool, streamed bool)
	// OnExecStream delivers [hiltypes.ExecStreamStart] and [hiltypes.ExecStreamLine] when streaming is used; nil disables streaming.
	OnExecStream func(any)
	// UIEvents optional; used with [CommandExecutionState] for [EXECUTING] chrome during run.
	UIEvents chan<- any
	// ExecCancelHub optional; ESC during [EXECUTING] cancels the command context (not the whole LLM turn).
	ExecCancelHub *execcancel.Hub

	// ExecutorProvider returns the current executor (local or remote). When nil, a local executor is used.
	ExecutorProvider func() execenv.CommandExecutor

	// OfflineMode when true: skip allowlist and executor; use RequestOfflinePaste instead of approval+run.
	OfflineMode func() bool
	// RequestOfflinePaste blocks until the user pastes output or cancels (offline mode only).
	RequestOfflinePaste func(command, reason, riskLevel string) hiltypes.OfflinePasteResponse
	// OnRemoteIssue, when non-nil, is informed about SSH transport errors and cleared on successful remote execution.
	OnRemoteIssue func(issue string)
}

var _ tool.InvokableTool = (*ExecuteCommandTool)(nil)

func (t *ExecuteCommandTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "execute_command",
		Desc: "Execute a shell command or script in the user's environment. Prefer shell commands to accomplish tasks; use Python or other scripting only when shell is not sufficient. Prefer pipelines and filters (grep, awk, jq/jsonpath, head/tail, etc.) so stdout contains only the information you need. stdout/stderr larger than 64 KiB are middle-truncated before they are shown to the model or stored in session history: the start and end are kept, and the omitted middle is replaced with a truncation notice. If the command is not on the allowlist, the tool waits for user approval before running. If output may contain secrets, set result_contains_secrets to true: the transcript is minimized for the user, the result is not stored in session history, and the model still receives stdout/stderr with heuristic redaction (patterns like tokens, JWTs, labeled passwords)—redaction is not guaranteed complete, so avoid printing raw secrets in commands.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     schema.String,
				Desc:     "Full command or script to run (may include pipes, etc.)",
				Required: true,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "Brief: why this command and expected effect. Must use the same language as the user's current question or instruction; do not switch to another language.",
				Required: false,
			},
			"risk_level": {
				Type:     schema.String,
				Desc:     "Risk level: read_only (no side effects), low (e.g. read config), high (e.g. restart, delete). Used for approval UI only.",
				Required: false,
			},
			"result_contains_secrets": {
				Type:     schema.Boolean,
				Desc:     "Set to true if the command output may contain secrets or private data. When true, the result is not stored in session history; the user sees a short redacted transcript; the model receives the same stdout/stderr shape with heuristic redaction (not a cryptographic guarantee). Large stdout/stderr are still middle-truncated to 64 KiB with head/tail preserved.",
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
	if riskLevel != "" && riskLevel != hiltypes.RiskLevelReadOnly && riskLevel != hiltypes.RiskLevelLow && riskLevel != hiltypes.RiskLevelHigh {
		riskLevel = "" // invalid value treated as not provided
	}
	sensitive := input.ResultContainsSecrets

	if t.OfflineMode != nil && t.OfflineMode() {
		return t.invokableRunOffline(ctx, command, reason, riskLevel, sensitive)
	}

	allowed := false
	if t.Allowlist != nil {
		allowed = t.Allowlist.CommandAllowsAutoApprove(command)
	}
	if !allowed {
		var autoHL []hiltypes.AutoApproveHighlightSpan
		if t.Allowlist != nil {
			autoHL = t.Allowlist.CommandAutoApproveHighlight(command)
		}
		resp := t.RequestApproval(command, "", reason, riskLevel, "", autoHL)
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

	cmdCtx, unregCancel := withCommandCancel(t.ExecCancelHub, ctx)
	defer unregCancel()
	endUI := pushCommandExecutionUI(t.UIEvents)
	defer endUI()

	streamStart := hiltypes.ExecStreamStart{Allowed: allowed, Suggested: false, Direct: false}
	outStr, errStr, exitCode, err, useStream := runExecutorWithStream(cmdCtx, executor, command, t.OnExecStream, streamStart)
	if t.OnRemoteIssue != nil {
		var connErr *execenv.SSHConnectionError
		if errors.As(err, &connErr) {
			if connErr.ReconnectSuccess {
				t.OnRemoteIssue("")
			} else {
				t.OnRemoteIssue(connErr.Error())
			}
		} else if _, ok := executor.(*execenv.SSHExecutor); ok && err == nil {
			t.OnRemoteIssue("")
		}
	}
	cancelled := errors.Is(cmdCtx.Err(), context.Canceled) || errors.Is(err, context.Canceled)
	if storeResult && t.Session != nil && !cancelled {
		_ = t.Session.AppendCommandResult(command, outStr, errStr, exitCode)
	}
	if cancelled {
		// Empty tail: "Execution cancelled." is shown when the host handles Esc; avoid duplicate Delve-prefixed lines.
		if t.OnExec != nil {
			if useStream {
				t.OnExec(command, allowed, "", sensitive || !storeResult, false, false, true)
			} else {
				t.OnExec(command, allowed, "", sensitive || !storeResult, false, false, false)
			}
		}
		return "The command was cancelled.", nil
	}
	uiOutStr := history.TruncateToolOutput(outStr)
	uiErrStr := history.TruncateToolOutput(errStr)
	var resultForUI string
	if useStream {
		resultForUI = "exit_code: " + strconv.Itoa(exitCode)
		if err != nil && (exitCode == 0 || execenv.IsSSHConnectionError(err)) {
			resultForUI += "\nerror: " + history.TruncateToolOutput(err.Error())
		}
	} else {
		resultForUI = uiOutStr
		if uiErrStr != "" {
			resultForUI += "\nstderr:\n" + uiErrStr
		}
		resultForUI += "\nexit_code: " + strconv.Itoa(exitCode)
		if err != nil && (exitCode == 0 || execenv.IsSSHConnectionError(err)) {
			resultForUI += "\nerror: " + history.TruncateToolOutput(err.Error())
		}
	}
	if t.OnExec != nil {
		t.OnExec(command, allowed, resultForUI, sensitive || !storeResult, false, false, useStream)
	}
	if sensitive {
		return history.RedactedToolResultMessage(outStr, errStr, exitCode, err), nil
	}
	return history.ToolResultMessage(outStr, errStr, exitCode, err), nil
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
	resultForUI := history.TruncateToolOutput(pasted)
	if resultForUI != "" {
		resultForUI += "\n\n" + manualPasteNoteForUI
	} else {
		resultForUI = manualPasteNoteForUI
	}
	if t.OnExec != nil {
		t.OnExec(command, false, resultForUI, sensitive || !storeResult, false, true, false)
	}
	if sensitive {
		if pasted == "" {
			return "The user submitted empty pasted output for: " + command + ".", nil
		}
		return "stdout:\n" + history.RedactAndTruncateToolOutput(pasted), nil
	}
	if pasted == "" {
		return "The user submitted empty pasted output for: " + command + ".", nil
	}
	return "stdout:\n" + history.TruncateToolOutput(pasted), nil
}
