// Package bootstrap wires feature packages that register TUI slash handlers, overlays, and message providers.
// Call [Install] once at process startup before constructing models that depend on those registries.
package bootstrap

import (
	"sync"

	"delve-shell/internal/configllm"
	"delve-shell/internal/remote"
	"delve-shell/internal/run"
	"delve-shell/internal/session"
	"delve-shell/internal/skill"
)

var installOnce sync.Once

// Install registers all interactive UI features. Safe to call multiple times (subsequent calls are no-ops).
func Install() {
	installOnce.Do(func() {
		configllm.Register()
		skill.Register()
		remote.Register()
		run.Register()
		session.Register()
	})
}
