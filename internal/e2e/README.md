# Terminal E2E Tests

PTY-driven tests that run the real `delve-shell` process and assert on terminal output.

## Run

```bash
go test ./internal/e2e/... -v -timeout=60s
```

Always set a **non-zero test timeout** (e.g. `-timeout=60s`); if a step stalls with **no new `step N: matched` log lines** for longer than the step’s `Expect` timeout, the binary or expectations are wrong—do not wait for the default `go test` timeout (10m).

- **Short mode**: `go test ./... -short` skips e2e (PTY/TUI tests are slow and environment-sensitive). Use this in CI or quick local runs.
- **Config**: The test writes a minimal `config.yaml` (with `llm.model: gpt-4o-mini`) under a temp root so the TUI starts without opening the Config LLM overlay.
- The cases below do not require LLM and pass as-is when run without `-short`.
- The approval-flow case (requires LLM) is skipped by default; to run it: `E2E_LLM=1 go test ./internal/e2e/... -v -run TUI_approval_flow`, with a valid LLM config on the machine.

## Cases overview

| Case name | Coverage |
|-----------|----------|
| TUI_smoke_help_quit | Startup, /help, /q |
| TUI_config_show | /config show: config path and LLM summary |
| TUI_cancel_no_request | /cancel with no in-flight request: prompt message |
| TUI_unknown_cmd | Invalid slash command (e.g. /foo): error message |
| TUI_run_direct | /run echo 1: direct run and result with exit_code |
| TUI_reload | /config reload: config and allowlist reload message |
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

## Troubleshooting

- **`/config show`, `/run`, `/help` behave like unknown command in e2e**: the real binary must **blank-import** feature packages that register slash handlers in `init()` (e.g. `internal/run`, `internal/remote`). The interactive entrypoint imports these alongside `internal/session`; if a new binary is added, mirror that import list or e2e will time out waiting for text that never appears.
- **Hang until `go test` global timeout**: check PTY read loop and step `Expect` strings; `ReadUntilAny` respects the step deadline and returns failure with a tail of captured output when nothing matches.
