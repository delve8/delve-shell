// Package e2e provides PTY-based terminal e2e tests.
//
// Cases are registered in TerminalCases in this file; add a Case to the slice to add a test.
// Run: go test ./internal/e2e/... -v
// LLM-dependent cases (e.g. TUI_approval_flow) are skipped by default; set E2E_LLM=1 to run them.
package e2e

import "time"

// Case describes one terminal e2e case: input sequence and expected output.
// Steps run in order: send Input, then ReadUntil any Expect substring (or timeout).
type Case struct {
	Name    string        // case name for t.Run
	Skip    string        // if non-empty, skip (e.g. "need E2E_LLM=1")
	Steps   []Step        // step list
	Timeout time.Duration // default per-step timeout; 0 => DefaultStepTimeout
}

// Step: send one line of input, then wait until output contains any Expect substring.
type Step struct {
	Input   string        // one line to send to PTY (\r\n added automatically)
	Expect  []string      // pass when output contains any; empty means no check
	Timeout time.Duration // this step's timeout; 0 => Case.Timeout
}

// DefaultStepTimeout is used when Case.Timeout is zero.
const DefaultStepTimeout = 8 * time.Second

// tuiReadyExpect: substrings that appear on the initial TUI (title line + placeholder). Main view uses titleLine() and KeyPlaceholderInput, not "delve-shell" or "Enter".
var tuiReadyExpect = []string{"Local", "IDLE", "Auto-Run", "Type", "slash"}

// TerminalCases is the registered list of terminal e2e cases; append to add cases.
var TerminalCases = []Case{
	{
		Name:    "TUI_smoke_help_quit",
		Skip:    "",
		Timeout: DefaultStepTimeout,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second}, // wait for TUI ready
			{Input: "/help", Expect: []string{"Slash commands", "斜杠命令", "Quit", "退出", "/q", "/run", "/help"}, Timeout: 5 * time.Second},
			{Input: "/q", Expect: []string{}, Timeout: 2 * time.Second},
		},
	},
	{
		Name:    "TUI_config_show",
		Skip:    "",
		Timeout: DefaultStepTimeout,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second},
			// KeyConfigHint EN: "Use /config llm for LLM; auto-run is in header." (header line also shows Auto-Run)
			{Input: "/config show", Expect: []string{"/config llm", "LLM", "auto-run", "header"}, Timeout: 5 * time.Second},
			{Input: "/q", Expect: []string{}, Timeout: 2 * time.Second},
		},
	},
	{
		Name:    "TUI_unknown_cmd",
		Skip:    "",
		Timeout: DefaultStepTimeout,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second},
			{Input: "/foo", Expect: []string{"Unknown command", "未知命令", "/q", "/run", "/help"}, Timeout: 5 * time.Second},
			{Input: "/q", Expect: []string{}, Timeout: 2 * time.Second},
		},
	},
	{
		Name:    "TUI_run_direct",
		Skip:    "",
		Timeout: DefaultStepTimeout,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second},
			{Input: "/run echo 1", Expect: []string{"Run:", "echo 1", "exit_code", "direct", "直接执行"}, Timeout: 5 * time.Second},
			{Input: "/q", Expect: []string{}, Timeout: 2 * time.Second},
		},
	},
	{
		Name:    "TUI_reload",
		Skip:    "",
		Timeout: DefaultStepTimeout,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second},
			{Input: "/config reload", Expect: []string{"reloaded", "重载", "Config and allowlist", "配置与允许列表"}, Timeout: 3 * time.Second},
			{Input: "/q", Expect: []string{}, Timeout: 2 * time.Second},
		},
	},
	{
		Name:    "TUI_approval_flow",
		Skip:    "need E2E_LLM=1 and valid LLM config",
		Timeout: 20 * time.Second,
		Steps: []Step{
			{Input: "", Expect: tuiReadyExpect, Timeout: 5 * time.Second},
			{Input: "Use execute_command to run `pwd` and then tell me the result.", Expect: []string{"Command to run", "待执行的命令", "1=Run", "1=approve", "2=Copy", "2=reject", "3=Dismiss"}, Timeout: 18 * time.Second},
			{Input: "1", Expect: []string{"exit_code", "Run:", "pwd"}, Timeout: 10 * time.Second},
		},
	},
}
