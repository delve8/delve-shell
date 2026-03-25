package hostapp

import (
	"sync"

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
	ShellSnapshot  chan<- []string
}

var (
	app struct {
		mu sync.RWMutex
		// send is the channel bundle installed by Wire (nil when unwired).
		send *Send
		// allowlistFn returns current allowlist auto-run for UI header and approval choice count; nil means "default on".
		allowlistFn func() bool
		// syncAllowlist persists allowlist_auto_run changes and invalidates runner (set from interactive startup).
		syncAllowlist func(bool)
		// remote mirrors header execution mode (updated when RemoteStatusMsg is processed).
		remoteActive bool
		remoteLabel  string
	}
	cfgLLMMu    sync.Mutex
	cfgLLMFirst bool
)

// Wire installs the send bundle for this process. The last call wins. Pass nil to clear (e.g. test cleanup).
func Wire(s *Send) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.send = s
}

func currentSend() *Send {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.send
}

// BindAllowlistAutoRun wires the allowlist auto-run getter and the sync callback invoked when /config changes it.
func BindAllowlistAutoRun(getter func() bool, sync func(bool)) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.allowlistFn = getter
	app.syncAllowlist = sync
}

// AllowlistAutoRunEnabled returns true when no getter is set, otherwise the getter result.
func AllowlistAutoRunEnabled() bool {
	app.mu.RLock()
	fn := app.allowlistFn
	app.mu.RUnlock()
	if fn == nil {
		return true
	}
	return fn()
}

// InvokeSyncAllowlistAutoRun runs the allowlist sync callback (persist + runner invalidate) when non-nil.
func InvokeSyncAllowlistAutoRun(v bool) {
	app.mu.RLock()
	fn := app.syncAllowlist
	app.mu.RUnlock()
	if fn != nil {
		fn(v)
	}
}

// SetRemoteExecution updates whether commands run on a remote executor and the header label.
func SetRemoteExecution(active bool, label string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.remoteActive = active
	app.remoteLabel = label
}

// RemoteActive reports whether the UI should treat execution as remote.
func RemoteActive() bool {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.remoteActive
}

// RemoteLabel returns the remote display label (e.g. "dev (user@host)"); empty when local or unset.
func RemoteLabel() string {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.remoteLabel
}

// SetOpenConfigLLMOnFirstLayout arms the next first-layout open (typically once per tea.Program from CLI).
func SetOpenConfigLLMOnFirstLayout(v bool) {
	cfgLLMMu.Lock()
	defer cfgLLMMu.Unlock()
	cfgLLMFirst = v
}

// TakeOpenConfigLLMOnFirstLayout returns whether to run startup overlay providers and clears the flag.
func TakeOpenConfigLLMOnFirstLayout() bool {
	cfgLLMMu.Lock()
	defer cfgLLMMu.Unlock()
	v := cfgLLMFirst
	cfgLLMFirst = false
	return v
}

// ResetTestState clears send wiring, allowlist bindings, remote mirror, and config-LLM one-shot. For tests only.
func ResetTestState() {
	app.mu.Lock()
	app.send = nil
	app.allowlistFn = nil
	app.syncAllowlist = nil
	app.remoteActive = false
	app.remoteLabel = ""
	app.mu.Unlock()
	cfgLLMMu.Lock()
	cfgLLMFirst = false
	cfgLLMMu.Unlock()
}

// Submit sends user text to the host controller (blocking). Returns false if unwired.
func Submit(text string) bool {
	s := currentSend()
	if s == nil || s.Submit == nil {
		return false
	}
	s.Submit <- text
	return true
}

// TrySubmitNonBlocking sends without blocking; returns false if unwired or buffer full.
func TrySubmitNonBlocking(text string) bool {
	s := currentSend()
	if s == nil || s.Submit == nil {
		return false
	}
	select {
	case s.Submit <- text:
		return true
	default:
		return false
	}
}

// NotifyConfigUpdated signals config or allowlist change (non-blocking; drops if full).
func NotifyConfigUpdated() {
	s := currentSend()
	if s == nil || s.ConfigUpdated == nil {
		return
	}
	select {
	case s.ConfigUpdated <- struct{}{}:
	default:
	}
}

// PublishCancelRequest forwards /cancel to the host controller. Returns false if unwired or full.
func PublishCancelRequest() bool {
	s := currentSend()
	if s == nil || s.CancelRequest == nil {
		return false
	}
	select {
	case s.CancelRequest <- struct{}{}:
		return true
	default:
		return false
	}
}

// PublishShellSnapshot sends transcript lines for /sh return restore. Returns false if unwired or full.
func PublishShellSnapshot(msgs []string) bool {
	s := currentSend()
	if s == nil || s.ShellSnapshot == nil {
		return false
	}
	select {
	case s.ShellSnapshot <- msgs:
		return true
	default:
		return false
	}
}

// PublishExecDirect sends a direct execution command (blocking until accepted).
func PublishExecDirect(cmd string) {
	if cmd == "" {
		return
	}
	s := currentSend()
	if s == nil || s.ExecDirect == nil {
		return
	}
	s.ExecDirect <- cmd
}

// PublishRemoteOnTarget forwards a remote connect target. Returns false if unwired or full.
func PublishRemoteOnTarget(target string) bool {
	s := currentSend()
	if s == nil || s.RemoteOn == nil {
		return false
	}
	select {
	case s.RemoteOn <- target:
		return true
	default:
		return false
	}
}

// PublishRemoteOff requests switching back to the local executor. Returns false if unwired or full.
func PublishRemoteOff() bool {
	s := currentSend()
	if s == nil || s.RemoteOff == nil {
		return false
	}
	select {
	case s.RemoteOff <- struct{}{}:
		return true
	default:
		return false
	}
}

// PublishRemoteAuthResponse forwards SSH auth answers. Returns false if unwired or full.
func PublishRemoteAuthResponse(resp remoteauth.Response) bool {
	s := currentSend()
	if s == nil || s.RemoteAuthResp == nil {
		return false
	}
	select {
	case s.RemoteAuthResp <- resp:
		return true
	default:
		return false
	}
}
