# UI Architecture Acceptance (2026-03-25)

## Scope

This acceptance note covers the recent refactor series that moved UI write-side effects to action intents and removed direct `internal/ui` dependency on `host/*` packages.

Non-goals of this note:
- product behavior review
- UI style/wording review
- broad host/controller redesign outside UI boundary work

## Current Boundary Status

Accepted boundary outcomes:
- `internal/ui` has no direct imports from `internal/host/*`, `internal/runtime/*`, or service packages.
- UI write-side operations are emitted as `uivm.UIAction` intents and consumed in `internal/host/controller/ui_actions.go`.
- Slash relay contract used by UI flow is `internal/uitypes.SlashSubmitPayload` (UI-owned contract path).
- Remote execution display state (`Active`, `Label`) is stored in `ui.Model` and updated via remote UI messages, not host mirrors.
- UI startup and allowlist read-side access is narrowed to `ui.ReadModel`.

This matches the targeted direction: UI acts as render + interaction + intent emission, while controller owns execution routing.

## Findings (High to Low Risk)

1) `internal/ui/actions.go` currently provides many Host-shaped helper methods (`Submit`, `PublishRemoteOnTarget`, `NotifyConfigUpdated`, etc.)
- Risk: naming and API shape can reintroduce conceptual coupling to legacy host semantics.
- Root-cause pattern: compatibility-oriented migration kept familiar method names to reduce churn.

2) `internal/ui/model_blackbox_test.go` has grown into a large multi-scenario file (400+ lines)
- Risk: behavior regressions become harder to localize; fixture complexity hides intent.
- Root-cause pattern: incremental refactors accumulated new scenario branches in a single test module.

3) Controller action consumer centralization is increasing in `internal/host/controller/ui_actions.go`
- Risk: continued growth can create a command-router hotspot and reduce change isolation.
- Root-cause pattern: fast consolidation of UI intent handling into one switch.

4) Remaining alias path exists for slash submit payload (`internal/host/route` alias to `internal/uitypes`)
- Risk: dual import paths may confuse future contributors during new feature work.
- Root-cause pattern: staged migration strategy retained compatibility shim.

## Ordered Refactor Plan (Single-purpose, Verifiable)

1. Rename UI action helper surface to intent-oriented names
- Example direction: `EmitSubmit`, `EmitRemoteOnTarget`, `EmitConfigUpdated`.
- Verification: `rg "func \\(m Model\\) (Submit|Publish|Notify|TrySubmit)" internal/ui` returns only intentionally kept compatibility methods (or none, per target).

2. Split blackbox scenarios by concern
- Candidate files: lifecycle/slash-commands/remote-overlay/startup-overlay.
- Verification: each test file stays focused on one scenario family; existing `go test ./internal/ui` remains green.

3. Split controller UI action handling by topic
- Candidate partitions: submit+lifecycle, remote, slash tracing/relay, config/allowlist.
- Verification: each handler file has clear ownership and no behavior change (`go test ./internal/host/controller`).

4. Remove slash payload alias shim after call-site cleanup
- Replace `internal/host/route` payload references with `internal/uitypes` everywhere.
- Verification: no remaining alias dependency by `rg "host/route\\.SlashSubmitPayload|type SlashSubmitPayload ="`.

## Regression Checklist

- `go test ./...` passes.
- Slash flows still work:
  - exact command dispatch
  - prefix dispatch
  - relay-to-UI path with selected index and input line
- Session commands still work:
  - `/new`
  - `/sessions <id>`
- Remote flows still work:
  - `/remote on`, `/remote off`
  - auth prompt response submit path
  - remote completion cache filtering by label
- Config update side effects still propagate:
  - allowlist auto-run toggle sync
  - config reload notifications

## Acceptance Decision

Accepted for current phase.

Rationale:
- Architectural goal for UI/package boundary has been met in the current code.
- Residual items are primarily maintainability hardening, not boundary correctness blockers.
