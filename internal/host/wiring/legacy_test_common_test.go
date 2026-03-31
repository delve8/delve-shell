package wiring

import (
	"testing"

	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/cmd"
)

// bindTestPorts wires hostapp send endpoints. Do not use t.Parallel().
func bindTestPorts(t *testing.T, ports bus.InputPorts, shell chan<- hostcmd.ShellSnapshot) *app.Runtime {
	t.Helper()
	rt := app.NewRuntime()
	t.Cleanup(func() { rt.Reset() })
	BindSendPorts(rt, ports, shell)
	return rt
}
