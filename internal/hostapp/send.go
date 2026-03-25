package hostapp

import (
	"delve-shell/internal/hostroute"
	"delve-shell/internal/remoteauth"
)

// Send is the narrowed send-side view of channels that feed hostbus.BridgeInputs.
type Send struct {
	Submit         chan<- string
	ConfigUpdated  chan<- struct{}
	CancelRequest  chan<- struct{}
	ExecDirect     chan<- string
	RemoteOn       chan<- string
	RemoteOff      chan<- struct{}
	RemoteAuthResp chan<- remoteauth.Response
	// SlashRequest receives a line immediately before the TUI runs a matched slash handler.
	SlashRequest chan<- string
	// SlashTrace receives slash lines after successful TUI dispatch (observability / future routing).
	SlashTrace chan<- string
	// SlashSubmit receives structured main-Enter slash intent for bus → controller → TUI relay (§10.8.1).
	SlashSubmit   chan<- hostroute.SlashSubmitPayload
	ShellSnapshot chan<- []string
}
