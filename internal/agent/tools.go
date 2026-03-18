package agent

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/skills"
)

// ApprovalResponse is the user's choice for a pending command: Run, Reject, or Copy (copy to clipboard, do not run).
type ApprovalResponse struct {
	Approved     bool // true = run the command
	CopyRequested bool // true = user chose Copy (do not run; copy to clipboard)
}

// ApprovalRequest is sent to HIL: pending command and response channel.
type ApprovalRequest struct {
	Command    string // command to run
	Summary    string // optional short summary (e.g. from SKILL.md); shown separately from Reason
	Reason     string // AI explanation (why, expected effect); may be empty
	RiskLevel  string // read_only | low | high; empty if not provided
	SkillName  string // non-empty when pending command is from run_skill (shown on approval card)
	ResponseCh chan ApprovalResponse
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

// ExecuteCommandTool runs a command/script; blocks on requestApproval until user chooses Run/Reject/Copy when not auto-run.
// When command may access sensitive path(s), blocks on requestSensitiveConfirmation for user to choose: refuse / run+store / run+no store.
// AllowlistAutoRun: when true, allowlisted commands run directly and only others show card (2 options: Run, Reject); when false, every command shows card (3 options: Run, Copy, Dismiss).
type ExecuteCommandTool struct {
	AllowlistAutoRun bool // when false, no command auto-runs; card has Run/Copy/Dismiss
	Allowlist        *hil.Allowlist
	SensitiveMatcher            *hil.SensitiveMatcher
	RequestApproval             func(command, summary, reason, riskLevel, skillName string) ApprovalResponse
	RequestSensitiveConfirmation func(command string) SensitiveChoice
	Session                     *history.Session
	OnExec                      func(command string, allowed bool, result string, sensitive bool, suggested bool)

	// ExecutorProvider returns the current executor (local or remote). When nil, a local executor is used.
	ExecutorProvider func() execenv.CommandExecutor
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

	// When AllowlistAutoRun is false, no command runs without user choice; when true, allowlist matches run directly.
	allowed := false
	if t.AllowlistAutoRun && t.Allowlist != nil {
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

// ListSkillsTool lists all installed skills (name, description only). Use get_skill to read one skill's full SKILL.md.
type ListSkillsTool struct{}

var _ tool.InvokableTool = (*ListSkillsTool)(nil)

func (t *ListSkillsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_skills",
		Desc: "List all installed skills under ~/.delve-shell/skills/. Returns each skill's name and description. Use get_skill(skill_name) to read one skill's full SKILL.md (usage, params, examples) before calling run_skill.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *ListSkillsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	list, err := skills.List()
	if err != nil {
		return "list_skills failed: " + err.Error(), nil
	}
	if len(list) == 0 {
		return "No skills installed. Skills live under ~/.delve-shell/skills/ (each subdir with SKILL.md and optional scripts/).", nil
	}
	var b strings.Builder
	b.WriteString("Installed skills:\n")
	for _, s := range list {
		b.WriteString("- ")
		b.WriteString(s.Name)
		b.WriteString(": ")
		b.WriteString(s.Description)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// GetSkillTool returns one skill's full detail and SKILL.md content so the AI can learn how to call run_skill.
type GetSkillTool struct{}

var _ tool.InvokableTool = (*GetSkillTool)(nil)

func (t *GetSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_skill",
		Desc: "Get one skill's full detail: description, script list, and complete SKILL.md content (usage, params, examples). Call this before run_skill so you know which script and args to use. Use list_skills first to see available skill names.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "Skill name (directory under ~/.delve-shell/skills/).",
				Required: true,
			},
		}),
	}, nil
}

func (t *GetSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input struct {
		SkillName string `json:"skill_name"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "get_skill requires skill_name", nil
	}
	skillName := strings.TrimSpace(input.SkillName)
	if skillName == "" {
		return "get_skill requires skill_name. Use list_skills to see available skills.", nil
	}

	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return "Skill not found: " + skillName + ". Use list_skills to see available skills, then get_skill(skill_name) to read its SKILL.md.", nil
	}
	meta, err := skills.LoadSKILL(skillDir)
	if err != nil {
		return "Failed to load skill: " + err.Error(), nil
	}
	scriptNames, err := skills.ListScripts(skillDir)
	if err != nil {
		scriptNames = nil
	}
	var b strings.Builder
	b.WriteString("Skill: ")
	b.WriteString(meta.Name)
	b.WriteString("\nDescription: ")
	b.WriteString(meta.Description)
	if len(scriptNames) == 0 {
		b.WriteString("\nScripts: (none)")
	} else {
		b.WriteString("\nScripts: ")
		b.WriteString(strings.Join(scriptNames, ", "))
	}
	b.WriteString("\nUsage: run_skill(skill_name=\"" + meta.Name + "\", script_name=\"<script>\", args=[...])")
	content, err := skills.ReadSKILLContent(skillDir)
	if err == nil && content != "" {
		b.WriteString("\n\n--- SKILL.md (full) ---\n")
		b.WriteString(content)
	}
	return b.String(), nil
}

// RunSkillTool runs a skill script via HIL approval; same approval/execution flow as execute_command.
type RunSkillTool struct {
	RequestApproval             func(command, summary, reason, riskLevel, skillName string) ApprovalResponse
	RequestSensitiveConfirmation func(command string) SensitiveChoice
	SensitiveMatcher            *hil.SensitiveMatcher
	Session                     *history.Session
	OnExec                      func(command string, allowed bool, result string, sensitive bool, suggested bool)
	ExecutorProvider            func() execenv.CommandExecutor
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

	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return "Skill not found: " + skillName + ". Use list_skills to see available skills, then get_skill(skill_name) to read its SKILL.md.", nil
	}
	if _, err := skills.ScriptPath(skillDir, scriptName); err != nil {
		return "Script not found in skill: " + scriptName + ". Use get_skill(skill_name=\"" + skillName + "\") to see scripts and SKILL.md.", nil
	}
	// Load metadata once for risk level, summary, scope, and potential remote upload directory.
	meta, _ := skills.LoadSKILL(skillDir)

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
	localScriptsDir := skills.ScriptsDir(skillDir)
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
		cmd, err = skills.BuildCommandInDir(remoteScriptsDir, scriptName, input.Args)
	} else {
		cmd, err = skills.BuildCommand(skillDir, scriptName, input.Args)
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
		case SensitiveRefuse:
			return "The user declined (sensitive path): " + cmd + ". Continue without running.", nil
		case SensitiveRunNoStore:
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

// syncSkillScriptsToRemote ensures that the local scriptsDir contents are present on the remote host under remoteScriptsDir.
// It compares remote and local file contents and only updates when they differ. No tar/gzip or extra tools are required;
// all operations use basic sh + mkdir + cat.
func syncSkillScriptsToRemote(ctx context.Context, executor execenv.CommandExecutor, scriptsDir, remoteScriptsDir string) error {
	if scriptsDir == "" || remoteScriptsDir == "" {
		return nil
	}
	if _, ok := executor.(*execenv.SSHExecutor); !ok {
		// Local executor: nothing to sync.
		return nil
	}
	info, err := os.Stat(scriptsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	// Ensure remote root directory exists.
	if _, _, _, err := executor.Run(ctx, "sh -c "+quoteForSh("mkdir -p "+remoteScriptsDir)); err != nil {
		return err
	}
	return filepath.WalkDir(scriptsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || rel == "." {
			return nil
		}
		localData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		remoteFile := remoteScriptsDir + "/" + rel
		remoteDir := remoteScriptsDir
		if idx := strings.LastIndex(remoteFile, "/"); idx > 0 {
			remoteDir = remoteFile[:idx]
		}
		// Read remote content if file exists.
		readCmd := "if [ -f " + quoteForSh(remoteFile) + " ]; then cat " + quoteForSh(remoteFile) + "; fi"
		remoteOut, _, _, _ := executor.Run(ctx, "sh -c "+quoteForSh(readCmd))
		if remoteOut == string(localData) {
			return nil
		}
		// Create parent dir and upload file via here-doc.
		uploadBuilder := &strings.Builder{}
		uploadBuilder.WriteString("mkdir -p ")
		uploadBuilder.WriteString(quoteForSh(remoteDir))
		uploadBuilder.WriteString(" && cat > ")
		uploadBuilder.WriteString(quoteForSh(remoteFile))
		// Use a delimiter that is very unlikely to appear in scripts.
		delimiter := "EOF_DELVE_SKILL"
		uploadBuilder.WriteString(" << '")
		uploadBuilder.WriteString(delimiter)
		uploadBuilder.WriteString("'\n")
		uploadBuilder.Write(localData)
		if !strings.HasSuffix(uploadBuilder.String(), "\n") {
			uploadBuilder.WriteString("\n")
		}
		uploadBuilder.WriteString(delimiter)
		uploadBuilder.WriteString("\n")
		if _, _, _, err := executor.Run(ctx, "sh -c "+quoteForSh(uploadBuilder.String())); err != nil {
			return err
		}
		return nil
	})
}

// quoteForSh wraps s in single quotes and escapes single quotes as '\''.
func quoteForSh(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
