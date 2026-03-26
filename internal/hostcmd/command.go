package hostcmd

import (
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

// Command is a structured host-side intent emitted by UI and consumed by controller.
type Command interface {
	hostCommand()
}

type Submission struct {
	Submission inputlifecycletype.InputSubmission
}

func (Submission) hostCommand() {}

type SessionNew struct{}

func (SessionNew) hostCommand() {}

type SessionSwitch struct {
	SessionID string
}

func (SessionSwitch) hostCommand() {}

type ExecDirect struct {
	Command string
}

func (ExecDirect) hostCommand() {}

type ConfigUpdated struct{}

func (ConfigUpdated) hostCommand() {}

type CancelRequested struct{}

func (CancelRequested) hostCommand() {}

type ShellSnapshot struct {
	Messages []string
}

func (ShellSnapshot) hostCommand() {}

type RemoteOnTarget struct {
	Target string
}

func (RemoteOnTarget) hostCommand() {}

type RemoteOff struct{}

func (RemoteOff) hostCommand() {}

type RemoteAuthReply struct {
	Response remoteauth.Response
}

func (RemoteAuthReply) hostCommand() {}

type AllowlistAutoRun struct {
	Enabled bool
}

func (AllowlistAutoRun) hostCommand() {}
