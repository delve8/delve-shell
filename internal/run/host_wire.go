package run

import "delve-shell/internal/hostapp"

// PublishCancelRequest forwards /cancel to the host controller when wired. Returns false if unwired or buffer full.
func PublishCancelRequest() bool { return hostapp.PublishCancelRequest() }

// PublishShellSnapshot sends transcript lines for /sh return restore. Returns false if unwired or buffer full.
func PublishShellSnapshot(msgs []string) bool { return hostapp.PublishShellSnapshot(msgs) }

// PublishExecDirect sends a direct execution command to the host controller (blocking until the channel accepts).
func PublishExecDirect(cmd string) { hostapp.PublishExecDirect(cmd) }
