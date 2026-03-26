<img src="assets/logo.svg" width="64" height="56" alt="delve-shell" />

# delve-shell

AI-assisted ops CLI with human-in-the-loop execution and auditable session history.

## What It Does

- Chat with an AI in the terminal to analyze ops tasks.
- Run commands only through the tool's approval boundary.
- Auto-run allowlisted read-only commands when enabled.
- Persist session history, approvals, and command results for audit.
- Support local and SSH-backed remote execution.

## Core Principles

- HIL first: command execution is gated by this tool, not by the model's wording.
- Auditable by default: session and execution flows are recorded as structured history.
- Clear boundaries: UI handles interaction and rendering; host-side orchestration owns execution and state transitions.
- Small feature modules: remote, run, skill, session, and config features register into shared contracts instead of hard-wiring into one giant entrypoint.

## Runtime Architecture

The interactive runtime is split into a few stable layers:

1. `cmd/delve-shell`
   Starts the CLI entrypoint.
2. `internal/cli/interactive`
   Wires the TUI, host bus, controller, runtime managers, and shutdown lifecycle.
3. `internal/ui`
   Bubble Tea shell: input box, transcript, overlays, approval cards, slash dropdown, and lifecycle result application.
4. `internal/host/controller`
   The orchestration core. It consumes structured host commands from UI and domain events from the bus, then coordinates runner, session, remote, and presenter flows.
5. `internal/host/bus`
   Domain event transport between input ports, controller, and UI presenter.
6. `internal/agent` and `internal/runtime/*`
   LLM/tool execution, executor management, runner management, and session/runtime coordination.

## Input And Command Flow

- User typing stays in `internal/ui`.
- Enter produces a structured `InputSubmission` through the unified input lifecycle.
- Chat submissions become `hostcmd.Submission` and are published as host bus events.
- Slash submissions go through the same lifecycle, then dispatch into feature-provided slash execution handlers.
- Control actions such as cancel, quit, and overlay close are handled as explicit control signals.

This means chat, slash, and control now share one submission model and one output model, instead of separate historical paths.

## Module Map

### UI and interaction

- `internal/ui`: Bubble Tea model, view, update routing, overlays, title bar, transcript rendering.
- `internal/uipresenter`: host-to-UI message adapter.
- `internal/uiflow/*`: small interaction helpers for approval and enter-flow behavior.
- `internal/uiregistry`: slash option providers that do not depend on `ui.Model`.

### Host orchestration

- `internal/host/app`: host-facing runtime facade used by the interactive shell.
- `internal/host/bus`: event kinds, event payloads, UI pump, and input bridges.
- `internal/host/controller`: event handlers and command handling.
- `internal/host/wiring`: runtime/bus/controller binding helpers.
- `internal/hostcmd`: structured commands emitted by UI and consumed by controller.

### Input lifecycle

- `internal/inputlifecycle`: submit router and lifecycle engine.
- `internal/inputpreflight`: pre-submit classification and slash-enter planning.
- `internal/inputprocess/*`: chat, slash, and control processors.
- `internal/inputlifecycletype`: shared lifecycle types, outputs, and payloads.
- `internal/inputoutput`: applies lifecycle results back into UI-facing state.

### Feature modules

- `internal/run`: direct `/run`, `/sh`, allowlist config helpers, local command completion.
- `internal/remote`: remote config, connect/disconnect, auth, and remote-specific UI state/events.
- `internal/skill`: skill install/update/remove, skill invocation, skill overlays.
- `internal/skillstore`: skill discovery, manifest parsing, install/update/remove, and source management.
- `internal/configllm`: LLM config overlay and config slash handling.
- `internal/session`: session switching and session-derived UI lines.
- `internal/bootstrap`: single explicit registration entrypoint for feature modules.

### Execution, safety, and persistence

- `internal/agent`: LLM runner and tools.
- `internal/hil` and `internal/hiltypes`: approval, allowlist, sensitive command checks, and related UI payloads.
- `internal/execenv`: local and SSH executors.
- `internal/history`: session history storage and replay.
- `internal/config`: config loading, writing, defaults, and path resolution.

## Slash And Overlay Design

- Slash suggestions are provider-based and intentionally lightweight.
- Slash execution is feature-registered through a single execution contract.
- Overlay-heavy features use a unified overlay feature contract for open, event, key, content, close, and startup hooks.
- Fill-only slash rows such as `/run <cmd>` are encoded as option metadata instead of separate legacy selected-handler registries.

This is intentionally plugin-like. The project currently does not assume a very large slash surface or a large number of feature modules.

## Config Paths

On first run, the app creates `config.yaml`, `allowlist.yaml`, and related files under a config root directory.

| Platform | Default config root |
|----------|---------------------|
| Linux    | `~/.delve-shell` |
| macOS    | `~/.delve-shell` |
| Windows  | `%USERPROFILE%\\.delve-shell` |

Override with:

```bash
export DELVE_SHELL_ROOT=/path/to/my-dir
```

Main files:

- Config: `<root>/config.yaml`
- Allowlist: `<root>/allowlist.yaml`
- Sessions: `<root>/sessions`
- Skills: `<root>/skills`

## Usage

1. Start: `./bin/delve-shell`
2. Enter a natural-language task or a slash command.
3. Approve non-allowlisted commands when prompted.
4. Review transcript, tool output, and session history in the same TUI.

## Open Work

Long-lived architecture notes and remaining cleanup items are tracked in `.cursor/architecture-open-items.md`.
