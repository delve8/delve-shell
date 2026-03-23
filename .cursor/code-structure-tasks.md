# Code structure improvements (delve-shell)

Tasks below are ordered **simple → complex**. Check off when done.

## P0 — Quick wins (cleanup / docs)

- [x] **Remove empty `internal/infra` tree**  
  `storage/`, `ssh/`, `git/` contained no `.go` files and nothing imported `internal/infra`. **Done:** directory removed.

- [x] **Clarify `hiltypes` vs `internal/hil`**  
  Package docs updated on `internal/agent/hiltypes` and `internal/agent/doc.go`: `hil` = policy; `hiltypes` = UI wire messages.

- [x] **Document thin `internal/service/*` packages**  
  **Done:** `configsvc/doc.go` added; package comments on `remotesvc`, `skillsvc` set to `// Package …` style.

## P1 — Medium effort (boundaries)

- [x] **`modelinfo` + submit path (document)**  
  **Done:** comment in `hostloop/submit.go` where `FetchModelContextLength` is used (best-effort HTTP, 0 => maxMsg-only).

- [ ] **`modelinfo` + submit path (behavior)**  
  Optional: add timeout wrapper, or move call behind `agent`/helper so `hostloop` does not own HTTP side effects.

- [ ] **Trim `cli.Run` surface**  
  Optionally extract “start TUI + wire `hostloop`” into a named helper or small type so `run.go` stays readable (no behavior change). `run.go` is already ~150 lines; defer until it grows again.

## P2 — Large refactor (do incrementally)

- [ ] **Split `internal/ui/model.go`** (~2.3k lines)  
  Continue extracting by feature: pending approval/sensitive, remote, config overlays, slash routing — match patterns already used for `overlay_*`, `config_handlers`, `slash.go`.  
  **Progress:** `openUpdateSkillOverlay` lives in `overlay_update_skill.go` with `handleUpdateSkillOverlayKey`.

- [ ] **Optional: `e2e` location**  
  If desired, move `internal/e2e` to repo-root `e2e/` or `test/e2e` for clearer separation from libraries (style only).

---

## Notes

- `consts` vs `config`: default prompts in `consts`; user overrides in `config` — acceptable unless product requires a single source of truth.
- Do not reintroduce empty directory placeholders without a `.go` file or a short README explaining intent.
