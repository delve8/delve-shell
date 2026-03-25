package app

import (
	"delve-shell/internal/host/route"
	"delve-shell/internal/remoteauth"
)

// Host is the injectable façade for host-side operations (bus sends, allowlist mirror, remote header mirror, config-LLM startup).
// *Runtime implements Host.
type Host interface {
	Submit(text string) bool
	TrySubmitNonBlocking(text string) bool
	NotifyConfigUpdated()
	PublishCancelRequest() bool
	PublishShellSnapshot(msgs []string) bool
	PublishExecDirect(cmd string)
	PublishRemoteOnTarget(target string) bool
	PublishRemoteOff() bool
	PublishRemoteAuthResponse(resp remoteauth.Response) bool
	BindAllowlistAutoRun(getter func() bool, sync func(bool))
	AllowlistAutoRunEnabled() bool
	InvokeSyncAllowlistAutoRun(v bool)
	SetRemoteExecution(active bool, label string)
	RemoteActive() bool
	RemoteLabel() string
	SetOpenConfigLLMOnFirstLayout(v bool)
	TakeOpenConfigLLMOnFirstLayout() bool
	// TryRelaySlashSubmit enqueues structured slash intent for controller → TUI relay; returns false if unwired or buffer full.
	TryRelaySlashSubmit(p route.SlashSubmitPayload) bool
	// RequestSlashDispatch records that the TUI is about to run a matched slash handler (non-blocking; drops if full).
	RequestSlashDispatch(line string)
	// TraceSlashEntered records a successfully dispatched slash line on the host bus (non-blocking; drops if full).
	TraceSlashEntered(line string)
}

// nopHost is a safe no-op Host for tests and idle processes.
type nopHost struct{}

func (nopHost) Submit(string) bool                                 { return false }
func (nopHost) TrySubmitNonBlocking(string) bool                   { return false }
func (nopHost) NotifyConfigUpdated()                               {}
func (nopHost) PublishCancelRequest() bool                         { return false }
func (nopHost) PublishShellSnapshot([]string) bool                 { return false }
func (nopHost) PublishExecDirect(string)                           {}
func (nopHost) PublishRemoteOnTarget(string) bool                  { return false }
func (nopHost) PublishRemoteOff() bool                             { return false }
func (nopHost) PublishRemoteAuthResponse(remoteauth.Response) bool { return false }
func (nopHost) BindAllowlistAutoRun(func() bool, func(bool))       {}
func (nopHost) AllowlistAutoRunEnabled() bool                      { return true }
func (nopHost) InvokeSyncAllowlistAutoRun(bool)                    {}
func (nopHost) SetRemoteExecution(bool, string)                    {}
func (nopHost) RemoteActive() bool                                 { return false }
func (nopHost) RemoteLabel() string                                { return "" }
func (nopHost) SetOpenConfigLLMOnFirstLayout(bool)                 {}
func (nopHost) TakeOpenConfigLLMOnFirstLayout() bool               { return false }
func (nopHost) TryRelaySlashSubmit(route.SlashSubmitPayload) bool  { return false }
func (nopHost) RequestSlashDispatch(string)                        {}
func (nopHost) TraceSlashEntered(string)                           {}

var (
	nopSingleton Host = nopHost{}
)

// Nop returns a no-op Host (same instance for all callers).
func Nop() Host { return nopSingleton }
