package wiring

import (
	"testing"

	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
)

// bindTestPorts wires hostapp send endpoints. Do not use t.Parallel().
func bindTestPorts(t *testing.T, ports bus.InputPorts, shell chan<- []string) *app.Runtime {
	t.Helper()
	rt := app.NewRuntime()
	t.Cleanup(func() { rt.Reset() })
	BindSendPorts(rt, ports, shell)
	return rt
}

// installTestRuntime returns an empty *Runtime for allowlist-only tests. Resets on cleanup.
func installTestRuntime(t *testing.T) *app.Runtime {
	t.Helper()
	rt := app.NewRuntime()
	t.Cleanup(func() { rt.Reset() })
	return rt
}
