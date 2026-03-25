package uivm

import "delve-shell/internal/host/route"

type UIActionKind string

const (
	UIActionSubmit            UIActionKind = "submit"
	UIActionRelaySlashSubmit  UIActionKind = "relay_slash_submit"
	UIActionRequestSlashTrace UIActionKind = "request_slash_trace"
	UIActionEnterSlashTrace   UIActionKind = "enter_slash_trace"
)

// UIAction is an outbound intent emitted by UI and consumed by controller.
type UIAction struct {
	Kind UIActionKind

	Text        string
	SlashSubmit route.SlashSubmitPayload
}
