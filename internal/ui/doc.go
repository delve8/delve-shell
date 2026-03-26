// Package ui implements the Bubble Tea TUI shell: model, view, update routing, and slash/provider registration.
//
// Architecture boundaries (target direction):
//   - [delve-shell/internal/uiflow/enterflow]: main Enter / slash relay and post-dispatch classification helpers.
//   - [delve-shell/internal/uiflow/approvalexec]: HIL decision → channel / clipboard side-effect mapping.
//   - [delve-shell/internal/uiregistry]: slash suggestion provider chains that do not depend on [Model].
//
// This package should avoid growing new business rules; prefer extending the packages above or feature modules
// that register via [RegisterSlashExecutionProvider], [RegisterOverlayFeature], and [RegisterStateEventProvider].
package ui
