package app

import (
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/remote/auth"
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
	ShellSnapshot  chan<- hostcmd.ShellSnapshot
}
