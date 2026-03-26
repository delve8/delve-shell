package uivm

import (
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

type UIActionKind string

const (
	UIActionSubmission       UIActionKind = "submission"
	UIActionSessionNew       UIActionKind = "session_new"
	UIActionSessionSwitch    UIActionKind = "session_switch"
	UIActionExecDirect       UIActionKind = "exec_direct"
	UIActionConfigUpdated    UIActionKind = "config_updated"
	UIActionCancelRequested  UIActionKind = "cancel_requested"
	UIActionShellSnapshot    UIActionKind = "shell_snapshot"
	UIActionRemoteOnTarget   UIActionKind = "remote_on_target"
	UIActionRemoteOff        UIActionKind = "remote_off"
	UIActionRemoteAuthReply  UIActionKind = "remote_auth_reply"
	UIActionAllowlistAutoRun UIActionKind = "allowlist_auto_run"
)

// UIAction is an outbound intent emitted by UI and consumed by controller.
type UIAction struct {
	Kind UIActionKind

	Text            string
	BoolValue       bool
	Messages        []string
	RemoteAuthReply remoteauth.Response
	Submission      inputlifecycletype.InputSubmission
}
