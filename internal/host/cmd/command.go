package hostcmd

import (
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/remote/auth"
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
	// InputHistory is local submitted-line recall (Up/Down); persisted across /bash TUI restart.
	InputHistory []string
	Mode         SubshellMode
}

func (ShellSnapshot) hostCommand() {}

type AccessRemote struct {
	Target     string
	Socks5Addr string
}

func (AccessRemote) hostCommand() {}

type AccessLocal struct{}

func (AccessLocal) hostCommand() {}

// AccessOffline selects offline (manual relay) mode: no in-process execution; execute_command uses paste-back HIL.
type AccessOffline struct{}

func (AccessOffline) hostCommand() {}

type RemoteAuthReply struct {
	Response remoteauth.Response
}

func (RemoteAuthReply) hostCommand() {}
