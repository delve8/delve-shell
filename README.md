<img src="assets/logo.png" width="64" height="64" alt="delve-shell" />

# delve-shell

AI-assisted ops CLI with human-in-the-loop execution and auditable session history.

## What It Does

- Chat with an AI in the terminal to analyze ops tasks.
- Run commands only through the tool's approval boundary.
- Auto-run allowlisted read-only commands when enabled.
- Persist session history, approvals, and command results for audit.
- Support local and SSH-backed remote execution.
- Maintain per-host memory so later turns can reuse stable machine facts and command availability.

## Core Principles

- HIL first: command execution is gated by this tool, not by the model's wording.
- Auditable by default: session and execution flows are recorded as structured history.
- Clear boundaries: UI handles interaction and rendering; host-side orchestration owns execution and state transitions.
- Feature packages register into shared contracts instead of hard-wiring into one giant entrypoint.

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
   LLM/tool execution, executor management, runner management, and session coordination (`executormgr`, `runnermgr`, `sessionmgr`).
7. `internal/run`
   Registers `/exec`, `/bash`, and related slash/UI hooks into `internal/ui` via `bootstrap` (feature wiring, not the process managers above).

## Input And Command Flow

- User typing stays in `internal/ui`.
- Enter yields a structured `InputSubmission` from `internal/input/lifecycletype`, routed by `internal/input/lifecycle` and processors under `internal/input/process/*`.
- Chat lines are sent to the host as `hostcmd.Submission` (package `hostcmd`, import path `internal/host/cmd`) on the command channel and become bus events.
- Slash lines share the same lifecycle, then run through registered execution handlers; `internal/slash/dispatch` covers behavior after exact/prefix routing misses.
- Cancel, quit, and overlay-close paths use explicit control signals from the lifecycle types.

Chat, slash, and control share one submission model and one output-application path (`internal/input/output`), instead of separate legacy pipelines.

## Module Map

### CLI

- `internal/cli/interactive`: wires the Bubble Tea program, preflight, host stack, and shutdown.
- `internal/cli/hostfsm`: finite state machine for interactive startup transitions.

### UI and interaction

- `internal/ui`: Bubble Tea model, view, update routing, overlays, title bar, transcript rendering, and lifecycle result application.
- `internal/ui/widget`: reusable TUI widgets (e.g. approval card, title bar, lists).
- `internal/ui/uivm`: transcript-oriented view-model types shared with host and presenter layers.
- `internal/ui/presenter`: host-to-UI presenter boundary.
- `internal/ui/flow/*`: small interaction helpers for approval execution mapping and enter-flow planning.
- `internal/ui/registry`: slash option providers that do not depend on `ui.Model`.
- `internal/i18n`: localized copy and help strings for the shell.

### Host orchestration

- `internal/host/app`: host-facing runtime facade used by the interactive shell.
- `internal/host/bus`: event kinds, event payloads, UI pump, and input bridges.
- `internal/host/controller`: event handlers and command handling.
- `internal/host/wiring`: runtime/bus/controller binding helpers.
- `internal/host/cmd`: structured commands emitted by UI and consumed by controller.

### Input lifecycle

- `internal/input/lifecycle`: submit router and lifecycle engine.
- `internal/input/preflight`: pre-submit classification and slash-enter planning.
- `internal/input/process/*`: chat, slash, and control processors.
- `internal/input/lifecycletype`: shared lifecycle types, outputs, and payloads.
- `internal/input/output`: applies lifecycle results back into UI-facing state.
- `internal/input/maininput`: main Enter planning for slash-aware lines (package `maininput`).

### Slash and small utilities

- `internal/slash/view`: slash suggestion rows, selection, and prefix filtering.
- `internal/slash/flow`: main Enter and early-Enter behavior for `/…` lines.
- `internal/slash/dispatch`: glue after exact/prefix slash routing misses.
- `internal/slash/reg`: generic ordered provider chain helper.
- `internal/pathcomplete`: TAB-completion state for overlay path fields (remote and skill overlays).
- `internal/textwrap`: width-aware wrapping for transcript and related UI.

### Feature Modules

- `internal/run`: direct `/exec`, `/bash`, allowlist config helpers, local command completion.
- `internal/remote`: remote config, connect/disconnect, auth, and remote-specific UI state/events.
- `internal/skill`: skill install/update/remove, skill invocation, skill overlays.
- `internal/skill/store`: skill discovery, manifest parsing, install/update/remove, and source management.
- `internal/skill/git`: shallow clone/fetch helpers for skill installs from git remotes (`package git`).
- `internal/config/llm`: model config overlay, default system/offline prompt text, OpenAI-compatible model context lookup, and config slash handling.
- `internal/history/tui`: `/history` slash options, transcript line mapping from stored events, and active session path for the picker (`package historytui`). Session file lifecycle stays in `internal/history` and `internal/runtime/sessionmgr`.
- `internal/bootstrap`: single explicit registration entrypoint for feature modules.

### Execution, Safety, and Persistence

- `internal/agent`: LLM runner and tools.
- `internal/runtime/executormgr`: current local or SSH executor and remote credential flow.
- `internal/runtime/runnermgr`: agent runner wiring (config, HIL, history, executor provider).
- `internal/runtime/sessionmgr`: session coordination helpers used by the controller.
- `internal/hil`: allowlist and sensitive-command checks (core HIL helpers).
- `internal/hil/approvalflow`: maps approval-card keyboard input to decisions.
- `internal/hil/approvalview`: choice metadata, placeholders, and transcript line models for approval UI.
- `internal/hil/types`: structured payloads for pending approvals and sensitive confirmations (`package hiltypes`).
- `internal/remote/execenv`: local and SSH executors.
- `internal/hostmem`: persistent per-host memory, host identity/probe application, and LLM summary rendering.
- `internal/history`: session history storage and replay.
- `internal/config`: config loading, writing, defaults, path resolution, and rules-dir text aggregation for prompts (`LoadRules`).

## Slash And Overlay Design

- Slash suggestions are provider-based and intentionally lightweight.
- Slash execution is feature-registered through a single execution contract.
- Overlay-heavy features use a unified overlay feature contract for open, event, key, content, close, and startup hooks.
- Fill-only slash rows such as `/exec <cmd>` are encoded as option metadata instead of separate legacy selected-handler registries.

The project favors a small plugin surface. It currently does not assume a very large slash surface or a large number of feature modules.

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
- Rules (optional markdown snippets concatenated for prompts): `<root>/rules/`
- Sessions: `<root>/sessions`
- Host memory: `<root>/hosts`
- Skill store: `<root>/skills`

## Usage

1. Start the `delve-shell` binary.
2. Enter a natural-language task or a slash command.
3. Approve non-allowlisted commands when prompted.
4. Review transcript, tool output, and session history in the same TUI.

## Common Slash Commands

- `/access` opens the execution-target picker for saved hosts plus `/access New`, `/access Local`, and `/access Offline`.
- `/access Offline` switches to manual relay mode: commands are shown for you to run elsewhere, then you paste results back.
- `/config` opens config actions. The built-in entries are `/config remove-remote` and `/config model`.
- `/config remove-remote {host}` removes a saved remote host from config.
- `/skill` opens the installed-skill picker plus `/skill New`, `/skill Remove`, and `/skill Update`.
- `/skill {name} [text]` invokes an installed skill for the current turn.
- `/skill Remove {skill_name}` removes an installed skill.
- `/skill Update {skill_name}` updates an installed skill from its recorded source.
- `/history` opens the session picker and preview flow.
- `/exec {cmd}` runs a one-off command directly without going through the AI.

## Host Memory

- delve-shell keeps persistent host memory per execution environment under `<root>/hosts`.
- The controller probes the current local or remote target, resolves a host-memory context, and injects a compact summary into later LLM turns when available.
- The agent can read and update that memory through `view_host_memory` and `update_host_memory`.
- Host memory is meant for stable, reusable facts: machine role, responsibilities, capabilities, package managers, and commands that are reliably available or missing for the current user profile.
- It is a useful prior, not a guarantee. Fresh command output wins over remembered facts, and durable new observations should be written back.

## Transcript And History Behavior

- Closing an overlay triggers a full screen refresh: delve-shell clears the visible terminal content, then replays the recent transcript so the main shell returns to a clean, deterministic state.
- Replay is capped to the latest `100000` transcript lines. If the session is longer, the shell prints a temporary banner at the top explaining that older content was truncated from the replay.
- Older content is still preserved in session history. Use `/history` to inspect or switch sessions when you need transcript content that is older than the replay window.

## Skill Shortcuts

- Type `/skill` to open the installed-skill dropdown.
- The dropdown includes `/skill New`, `/skill Remove`, and `/skill Update` after installed skills.
- Title-case reserved rows stay distinct from real skills whose names are lowercase `new`, `remove`, or `update`.
