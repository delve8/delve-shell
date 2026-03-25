package hostwiring

import (
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/run"
)

// BindAllowlistAutoRun wires the allowlist auto-run getter (for UI header) and the sync callback invoked when /config changes it.
func BindAllowlistAutoRun(getter func() bool, sync func(bool)) {
	hostnotify.SetAllowlistAutoRunGetter(getter)
	run.SetSyncAllowlistAutoRun(sync)
}
