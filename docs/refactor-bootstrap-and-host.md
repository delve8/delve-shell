# Bootstrap, host layout, and related refactors

This document tracks the one-round tasks for explicit TUI registration, `internal/host/*` layout, observability, and small cleanups. Status reflects the tree at the time of the change.

## Task list

| # | Task | Status |
|---|------|--------|
| 1 | Add `internal/bootstrap.Install()` with `sync.Once`; replace blank `_` imports in `cli` / `interactive` / blackbox tests; move feature `init()` bodies to `Register()` in `configllm`, `skill`, `remote`, `run`, `session`. | done |
| 2 | Relocate `internal/host{bus,controller,app,route,wiring}` to `internal/host/{bus,controller,app,route,wiring}` with short package names (`bus`, `controller`, `app`, `route`, `wiring`). | done |
| 3 | Document `ui.Model.Update` message routing in `internal/ui/model.go`. | done |
| 4 | Add controller test: every `bus.Kind` constant has exactly one entry in `hostEventHandlers`. | done |
| 5 | Split `agent.RunnerOptions` into nested structs; update `runnermgr` construction. | done |
| 6 | Optional structured bus trace: when `DELVE_SHELL_TRACE_BUS=1`, log `Event.RedactedSummary()` from the controller loop. | done |
| 7 | Add `internal/ui` message factory functions; use them from `uipresenter` instead of struct literals. | done |

## Usage

- Call `bootstrap.Install()` once before constructing UI models that depend on slash or overlay registration (the interactive CLI does this at startup).
- For bus tracing during development: `DELVE_SHELL_TRACE_BUS=1`.

## Notes

- `bootstrap.Install` is idempotent (`sync.Once`) so tests may call it from `TestMain` without double-registering provider chains.
