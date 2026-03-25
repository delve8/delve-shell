package app

import (
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

// Send is the narrowed send-side view of channels that feed bus.BridgeInputs.
type Send struct {
	Submission     chan<- inputlifecycletype.InputSubmission
	ConfigUpdated  chan<- struct{}
	CancelRequest  chan<- struct{}
	ExecDirect     chan<- string
	RemoteOn       chan<- string
	RemoteOff      chan<- struct{}
	RemoteAuthResp chan<- remoteauth.Response
	// SlashRequest receives a line immediately before the TUI runs a matched slash handler.
	SlashRequest chan<- string
	// SlashTrace receives slash lines after successful TUI dispatch (observability / future routing).
	SlashTrace    chan<- string
	ShellSnapshot chan<- []string
}
