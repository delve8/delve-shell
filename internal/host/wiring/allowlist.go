package wiring

import "delve-shell/internal/host/app"

// BindAllowlistAutoRun wires the allowlist auto-run getter (for the UI footer/status bar) and the sync callback invoked when /config changes it.
func BindAllowlistAutoRun(r *app.Runtime, getter func() bool, sync func(bool)) {
	r.BindAllowlistAutoRun(getter, sync)
}
