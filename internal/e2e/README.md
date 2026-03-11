# Terminal E2E Tests

PTY-driven tests that run the real `delve-shell` process and assert on terminal output.

## Run

```bash
go test ./internal/e2e/... -v
```

- The cases below do not require LLM and pass as-is.
- The approval-flow case (requires LLM) is skipped by default; to run it: `E2E_LLM=1 go test ./internal/e2e/... -v -run TUI_approval_flow`, with a valid LLM config on the machine.

## Cases overview

| Case name | Coverage |
|-----------|----------|
| TUI_smoke_help_exit | Startup, /help, /exit |
| TUI_config_show | /config show: config path and LLM summary |
| TUI_cancel_no_request | /cancel with no in-flight request: prompt message |
| TUI_unknown_cmd | Invalid slash command (e.g. /foo): error message |
| TUI_run_direct | /run echo 1: direct run and result with exit_code |
| TUI_reload | /reload: config and allowlist reload message |
| TUI_approval_flow | Requires LLM: send message → approval card → y → result (skipped by default) |

## Test case management

- **Definition**: slice `TerminalCases` in `cases.go`.
- **Add a case**: append a `Case` to `TerminalCases`; no change to the test runner.

### Case fields

| Field | Description |
|-------|-------------|
| `Name` | Case name, used for `-run TestTerminalE2E/Name`. |
| `Skip` | If non-empty, the case is skipped by default (can be overridden in the test via env, e.g. `TUI_approval_flow`). |
| `Steps` | Step list: each step sends `Input` (one line, `\r\n` added), then waits until any `Expect` appears in output. |
| `Timeout` | Default per-step timeout; a step can set `Step.Timeout` to override. |

### Step fields

| Field | Description |
|-------|-------------|
| `Input` | One line sent to the PTY; empty means wait only, no send. |
| `Expect` | Pass when terminal output contains any of these substrings; empty means no check. |
| `Timeout` | Timeout for this step; 0 uses `Case.Timeout`. |

Terminal output is stripped of ANSI escapes before matching for stable assertions.
