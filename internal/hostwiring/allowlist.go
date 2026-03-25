package hostwiring

import "delve-shell/internal/hostapp"

// BindAllowlistAutoRun wires the allowlist auto-run getter (for UI header) and the sync callback invoked when /config changes it.
func BindAllowlistAutoRun(r *hostapp.Runtime, getter func() bool, sync func(bool)) {
	r.BindAllowlistAutoRun(getter, sync)
}
