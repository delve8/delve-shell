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

// HistoryPreviewOpen asks the host to show a read-only history preview; the user confirms with Enter in the overlay.
type HistoryPreviewOpen struct {
	SessionID string
}

func (HistoryPreviewOpen) hostCommand() {}

type ExecDirect struct {
	Command string
}

func (ExecDirect) hostCommand() {}

type ConfigUpdated struct{}

func (ConfigUpdated) hostCommand() {}

type CancelRequested struct{}

func (CancelRequested) hostCommand() {}

// SubshellMode selects how /bash behaves after the TUI exits.
type SubshellMode int

const (
	// SubshellModeLocalBash runs bash -i on stdio (default).
	SubshellModeLocalBash SubshellMode = iota
	// SubshellModeRemoteSSH runs an interactive shell over the existing SSH client (no second dial).
	SubshellModeRemoteSSH
)

type ShellSnapshot struct {
	Messages []string
	Mode     SubshellMode
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
