package app

import (
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

// Host is the injectable façade for host-side operations (bus sends, remote footer/status mirror, config-LLM startup).
// *Runtime implements Host.
type Host interface {
	SubmitSubmission(sub inputlifecycletype.InputSubmission) bool
	TrySubmitSubmissionNonBlocking(sub inputlifecycletype.InputSubmission) bool
	NotifyConfigUpdated()
	PublishCancelRequest() bool
	PublishShellSnapshot(snap hostcmd.ShellSnapshot) bool
	PublishExecDirect(cmd string)
	PublishRemoteOnTarget(target string) bool
	PublishRemoteOff() bool
	PublishRemoteAuthResponse(resp remoteauth.Response) bool
	SetRemoteExecution(active bool, label string)
	RemoteActive() bool
	RemoteLabel() string
	SetOpenConfigLLMOnFirstLayout(v bool)
	TakeOpenConfigLLMOnFirstLayout() bool
}

// nopHost is a safe no-op Host for tests and idle processes.
type nopHost struct{}

func (nopHost) SubmitSubmission(inputlifecycletype.InputSubmission) bool { return false }
func (nopHost) TrySubmitSubmissionNonBlocking(inputlifecycletype.InputSubmission) bool {
	return false
}
func (nopHost) NotifyConfigUpdated()                               {}
func (nopHost) PublishCancelRequest() bool                         { return false }
func (nopHost) PublishShellSnapshot(hostcmd.ShellSnapshot) bool { return false }
func (nopHost) PublishExecDirect(string)                           {}
func (nopHost) PublishRemoteOnTarget(string) bool                  { return false }
func (nopHost) PublishRemoteOff() bool                             { return false }
func (nopHost) PublishRemoteAuthResponse(remoteauth.Response) bool { return false }
func (nopHost) SetRemoteExecution(bool, string)                    {}
func (nopHost) RemoteActive() bool                                 { return false }
func (nopHost) RemoteLabel() string                                { return "" }
func (nopHost) SetOpenConfigLLMOnFirstLayout(bool)                 {}
func (nopHost) TakeOpenConfigLLMOnFirstLayout() bool               { return false }

var (
	nopSingleton Host = nopHost{}
)

// Nop returns a no-op Host (same instance for all callers).
func Nop() Host { return nopSingleton }
