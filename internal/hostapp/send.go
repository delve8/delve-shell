package hostapp

import "delve-shell/internal/remoteauth"

// Send is the narrowed send-side view of channels that feed hostbus.BridgeInputs.
type Send struct {
	Submit         chan<- string
	ConfigUpdated  chan<- struct{}
	CancelRequest  chan<- struct{}
	ExecDirect     chan<- string
	RemoteOn       chan<- string
	RemoteOff      chan<- struct{}
	RemoteAuthResp chan<- remoteauth.Response
	// SlashTrace receives slash lines after TUI dispatch (observability / future routing).
	SlashTrace    chan<- string
	ShellSnapshot chan<- []string
}
