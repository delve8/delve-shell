package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/hiltypes"
	"delve-shell/internal/history"
	"delve-shell/internal/skillstore"
)

// RunSkillTool runs a skill script via HIL approval; same approval/execution flow as execute_command.
type RunSkillTool struct {
	RequestApproval              func(command, summary, reason, riskLevel, skillName string) hiltypes.ApprovalResponse
	RequestSensitiveConfirmation func(command string) hiltypes.SensitiveChoice
	SensitiveMatcher             *hil.SensitiveMatcher
	Session                      *history.Session
	OnExec                       func(command string, allowed bool, result string, sensitive bool, suggested bool)
	ExecutorProvider             func() execenv.CommandExecutor
}

var _ tool.InvokableTool = (*RunSkillTool)(nil)

func (t *RunSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "run_skill",
		Desc: "Run a script from an installed skill. Skills are under ~/.delve-shell/skills/<skill_name>/ with SKILL.md and scripts/ subdir. Use list_skills to discover skills and their scripts. This tool always shows an approval card; the command runs in the skill's scripts/ directory. Set result_contains_secrets if the script output may contain sensitive data.",
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
				Type:     schema.Array,
				Desc:     "Optional list of arguments to pass to the script.",
				Required: false,
			},
			"reason": {
				Type:     schema.String,
				Desc:     "Brief explanation for the approval card.",
				Required: false,
			},
			"risk_level": {
				Type:     schema.String,
				Desc:     "read_only, low, or high. Overrides skill default if set.",
				Required: false,
			},
			"result_contains_secrets": {
				Type:     schema.Boolean,
				Desc:     "Set true if output may contain secrets; result is not returned to the model or stored.",
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
		case "local":
			if isRemote {
				return "Skill " + skillName + " is local-only (scope=local); connect locally and retry.", nil
			}
		case "remote":
			if !isRemote {
				return "Skill " + skillName + " is remote-only (scope=remote); connect to a remote host and retry.", nil
			}
		}
	}
	if isRemote {
		cmd, err = skillstore.BuildCommandInDir(remoteScriptsDir, scriptName, input.Args)
	} else {
		cmd, err = skillstore.BuildCommand(skillDir, scriptName, input.Args)
	}
	if err != nil {
		return "Failed to build skill command: " + err.Error(), nil
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

	// Always request approval for run_skill (no allowlist shortcut).
	resp := t.RequestApproval(cmd, summary, reason, riskLevel, skillName)
	if t.Session != nil {
		_ = t.Session.AppendCommand(cmd, resp.Approved, reason, riskLevel, "skill", skillName)
	}
	if resp.CopyRequested {
		if t.Session != nil {
			_ = t.Session.AppendSuggestedCommand(cmd, reason, riskLevel, "skill", skillName)
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

	// For remote executors, sync local skill scripts/ to a remote temp dir before running.
	if isRemote && remoteScriptsDir != "" {
		if err := syncSkillScriptsToRemote(ctx, executor, localScriptsDir, remoteScriptsDir); err != nil {
			return "Failed to sync skill scripts to remote: " + err.Error(), nil
		}
	}
	outStr, errStr, exitCode, err := executor.Run(ctx, cmd)
	if storeResult && t.Session != nil {
		_ = t.Session.AppendCommandResult(cmd, outStr, errStr, exitCode)
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
		t.OnExec(cmd, false, resultForUI, sensitive, false)
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
