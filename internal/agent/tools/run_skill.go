package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/agent/ctx"
	"delve-shell/internal/hil"
	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/execcancel"
	"delve-shell/internal/skill/store"
)

// RunSkillTool runs a skill script via HIL approval by default; when the LLM context is a /skill <name> turn
// and skill_name matches, approval is skipped (sensitive-path confirmation still applies).
type RunSkillTool struct {
	RequestApproval              func(command, summary, reason, riskLevel, skillName string, autoApproveHL []hiltypes.AutoApproveHighlightSpan) hiltypes.ApprovalResponse
	RequestSensitiveConfirmation func(command string) hiltypes.SensitiveChoice
	SensitiveMatcher             *hil.SensitiveMatcher
	Session                      *history.Session
	OnExec                       func(command string, allowed bool, result string, sensitive bool, suggested bool, offlineManual bool, streamed bool)
	// OnExecStream optional; when set and executor supports streaming, same path as execute_command (live lines in transcript).
	OnExecStream     func(any)
	UIEvents         chan<- any
	ExecCancelHub    *execcancel.Hub
	ExecutorProvider func() execenv.CommandExecutor
	OnRemoteIssue    func(issue string)
}

var _ tool.InvokableTool = (*RunSkillTool)(nil)

func (t *RunSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_skill",
		Desc: "Run a script from an installed skill. Skills are under ~/.delve-shell/skills/<skill_name>/ with SKILL.md and scripts/ subdir. Use list_skills to discover skills and their scripts. When the user started the turn with /skill <name> for the same skill, approval is skipped; otherwise an approval card is shown. The command runs in the skill's scripts/ directory. Prefer scripts that print concise, task-relevant summaries rather than large raw dumps. stdout/stderr larger than 64 KiB are middle-truncated before they are shown to the model or stored in session history: the start and end are kept, and the omitted middle is replaced with a truncation notice. Set result_contains_secrets when output may include secrets: history is not stored for that run, the user sees a short redacted transcript, and the model receives redacted stdout/stderr (heuristic, not guaranteed).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "Skill name (directory under ~/.delve-shell/skills/).",
				Required: true,
			},
			"script_name": {
				Type:     schema.String,
				Desc:     "Script name (file under the skill's scripts/ directory, e.g. run.sh).",
				Required: true,
			},
			"args": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One argument passed to the script.",
				},
				Desc:     "Optional list of arguments to pass to the script.",
				Required: false,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "Brief: why this script and expected effect. Must use the same language as the user's current question or instruction; do not switch to another language.",
				Required: false,
			},
			"risk_level": {
				Type:     schema.String,
				Desc:     "read_only, low, or high. Overrides skill default if set.",
				Required: false,
			},
			"result_contains_secrets": {
				Type:     schema.Boolean,
				Desc:     "Set true if output may contain secrets. Result is not stored in session history; model receives redacted stdout/stderr (same shape as normal tool output). Large stdout/stderr are still middle-truncated to 64 KiB with head/tail preserved.",
				Required: false,
			},
		}),
	}, nil
}

func (t *RunSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input struct {
		SkillName             string   `json:"skill_name"`
		ScriptName            string   `json:"script_name"`
		Args                  []string `json:"args"`
		Reason                string   `json:"reason"`
		RiskLevel             string   `json:"risk_level"`
		ResultContainsSecrets bool     `json:"result_contains_secrets"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "run_skill requires skill_name and script_name", nil
	}
	skillName := strings.TrimSpace(input.SkillName)
	scriptName := strings.TrimSpace(input.ScriptName)
	if skillName == "" || scriptName == "" {
		return "run_skill requires skill_name and script_name", nil
	}
	if input.Args == nil {
		input.Args = nil
	}

	skillDir := skillstore.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return "Skill not found: " + skillName + ". Use list_skills to see available skills, then get_skill(skill_name) to read its SKILL.md.", nil
	}
	if _, err := skillstore.ScriptPath(skillDir, scriptName); err != nil {
		return "Script not found in skill: " + scriptName + ". Use get_skill(skill_name=\"" + skillName + "\") to see scripts and SKILL.md.", nil
	}
	// Load metadata once for risk level, summary, scope, and potential remote upload directory.
	meta, _ := skillstore.LoadSKILL(skillDir)

	// Determine executor up front so we know whether commands will run locally or on a remote host.
	executor := execenv.CommandExecutor(execenv.LocalExecutor{})
	if t.ExecutorProvider != nil {
		if e := t.ExecutorProvider(); e != nil {
			executor = e
		}
	}
	// When executor is SSH, scripts run on a remote host and must be synced first.
	_, isRemote := executor.(*execenv.SSHExecutor)

	// Decide working directory and shell command string used for approval and execution.
	localScriptsDir := skillstore.ScriptsDir(skillDir)
	var remoteScriptsDir string
	if isRemote {
		base := ""
		if meta != nil && strings.TrimSpace(meta.RemoteUploadDir) != "" {
			base = strings.TrimSpace(meta.RemoteUploadDir)
		}
		if base == "" {
			base = "/tmp"
		}
		base = strings.TrimRight(base, "/")
		if base == "" {
			base = "/tmp"
		}
		remoteScriptsDir = base + "/delve-shell-skills/" + skillName + "/scripts"
	}
	var cmd string
	var err error
	// Enforce scope: local / remote / both (empty => both).
	if meta != nil {
		scope := strings.TrimSpace(strings.ToLower(meta.Scope))
		switch scope {
		case skillstore.ScopeLocal:
			if isRemote {
				return "Skill " + skillName + " is local-only (scope=local); connect locally and retry.", nil
			}
		case skillstore.ScopeRemote:
			if !isRemote {
				return "Skill " + skillName + " is remote-only (scope=remote); connect to a remote host and retry.", nil
			}
		}
	}
	if isRemote {
		abs, absErr := filepath.Abs(remoteScriptsDir)
		if absErr != nil {
			return "Failed to build skill command: " + absErr.Error(), nil
		}
		cmd = "cd " + quoteForSh(abs) + " && bash " + quoteForSh(scriptName)
		for _, a := range input.Args {
			cmd += " " + quoteForSh(a)
		}
	} else {
		abs, absErr := filepath.Abs(skillstore.ScriptsDir(skillDir))
		if absErr != nil {
			return "Failed to build skill command: " + absErr.Error(), nil
		}
		cmd = "cd " + quoteForSh(abs) + " && bash " + quoteForSh(scriptName)
		for _, a := range input.Args {
			cmd += " " + quoteForSh(a)
		}
	}

	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "Run skill " + skillName + " script " + scriptName
	}
	summary := ""
	if meta != nil && strings.TrimSpace(meta.Summary) != "" {
		summary = strings.TrimSpace(meta.Summary)
	}
	// For run_skill, risk_level is determined solely by SKILL.md; ignore tool input risk_level.
	riskLevel := ""
	if meta != nil && meta.RiskLevel != "" {
		riskLevel = strings.TrimSpace(strings.ToLower(meta.RiskLevel))
	}

	var resp hiltypes.ApprovalResponse
	if slashSkill, ok := agentctx.SkillSlashSkillName(ctx); ok && strings.EqualFold(slashSkill, skillName) {
		resp = hiltypes.ApprovalResponse{Approved: true}
	} else {
		resp = t.RequestApproval(cmd, summary, reason, riskLevel, skillName, nil)
	}
	if t.Session != nil {
		_ = t.Session.AppendCommand(cmd, resp.Approved, reason, riskLevel, history.CommandPayloadKindSkill, skillName)
	}
	if resp.CopyRequested {
		if t.Session != nil {
			_ = t.Session.AppendSuggestedCommand(cmd, reason, riskLevel, history.CommandPayloadKindSkill, skillName)
		}
		return "The user copied the command and did not run it. Continue with your reply or suggest next steps.", nil
	}
	if !resp.Approved {
		return "The user declined to run this skill: " + skillName + " " + scriptName + ". Continue without running it.", nil
	}

	storeResult := true
	sensitive := input.ResultContainsSecrets
	if t.RequestSensitiveConfirmation != nil && t.SensitiveMatcher != nil && t.SensitiveMatcher.MayAccessSensitivePath(cmd) {
		choice := t.RequestSensitiveConfirmation(cmd)
		switch choice {
		case hiltypes.SensitiveRefuse:
			return "The user declined (sensitive path): " + cmd + ". Continue without running.", nil
		case hiltypes.SensitiveRunNoStore:
			storeResult = false
			sensitive = true
		}
	} else if input.ResultContainsSecrets {
		storeResult = false
		sensitive = true
	}

	cmdCtx, unregCancel := withCommandCancel(t.ExecCancelHub, ctx)
	defer unregCancel()

	// For remote executors, sync local skill scripts/ to a remote temp dir before running.
	// Keep [EXECUTING] off until sync finishes: SCP can take a while with no stdout yet, which looked like a stuck run.
	if isRemote && remoteScriptsDir != "" {
		sendAgentNotify(t.UIEvents, i18n.T(i18n.KeySkillScriptsSyncRemote))
		if err := syncSkillScriptsToRemote(cmdCtx, executor, localScriptsDir, remoteScriptsDir); err != nil {
			return "Failed to sync skill scripts to remote: " + err.Error(), nil
		}
	}

	endUI := pushCommandExecutionUI(t.UIEvents)
	defer endUI()
	streamStart := hiltypes.ExecStreamStart{Allowed: false, Suggested: false, Direct: false}
	outStr, errStr, exitCode, err, streamed := runExecutorWithStream(cmdCtx, executor, cmd, t.OnExecStream, streamStart)
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
		_ = t.Session.AppendCommandResult(cmd, outStr, errStr, exitCode)
	}
	if cancelled {
		if t.OnExec != nil {
			t.OnExec(cmd, false, "", sensitive, false, false, streamed)
		}
		return "The skill command was cancelled.", nil
	}
	uiOutStr := history.TruncateToolOutput(outStr)
	uiErrStr := history.TruncateToolOutput(errStr)
	var resultForUI string
	if streamed {
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
		t.OnExec(cmd, false, resultForUI, sensitive, false, false, streamed)
	}
	if sensitive {
		return history.RedactedToolResultMessage(outStr, errStr, exitCode, err), nil
	}
	return history.ToolResultMessage(outStr, errStr, exitCode, err), nil
}
