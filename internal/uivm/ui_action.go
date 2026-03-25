package uivm

import (
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/uitypes"
)

type UIActionKind string

const (
	UIActionSubmit            UIActionKind = "submit"
	UIActionExecDirect        UIActionKind = "exec_direct"
	UIActionConfigUpdated     UIActionKind = "config_updated"
	UIActionCancelRequested   UIActionKind = "cancel_requested"
	UIActionShellSnapshot     UIActionKind = "shell_snapshot"
	UIActionRemoteOnTarget    UIActionKind = "remote_on_target"
	UIActionRemoteOff         UIActionKind = "remote_off"
	UIActionRemoteAuthReply   UIActionKind = "remote_auth_reply"
	UIActionAllowlistAutoRun  UIActionKind = "allowlist_auto_run"
	UIActionRelaySlashSubmit  UIActionKind = "relay_slash_submit"
	UIActionRequestSlashTrace UIActionKind = "request_slash_trace"
	UIActionEnterSlashTrace   UIActionKind = "enter_slash_trace"
)

// UIAction is an outbound intent emitted by UI and consumed by controller.
type UIAction struct {
	Kind UIActionKind

	Text            string
	BoolValue       bool
	Messages        []string
	SlashSubmit     uitypes.SlashSubmitPayload
	RemoteAuthReply remoteauth.Response
}
