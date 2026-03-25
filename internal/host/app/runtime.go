package app

import (
	"sync"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

// Runtime holds host wiring and UI mirrors for one process (or one test fixture).
type Runtime struct {
	mu sync.RWMutex
	// send is the channel bundle installed by WireSend (nil when unwired).
	send *Send
	// allowlistFn returns current allowlist auto-run for UI header and approval choice count; nil means "default on".
	allowlistFn func() bool
	// syncAllowlist persists allowlist_auto_run changes and invalidates runner.
	syncAllowlist func(bool)
	remoteActive  bool
	remoteLabel   string
	cfgLLMMu      sync.Mutex
	cfgLLMFirst   bool
}

// NewRuntime returns an empty runtime; call WireSend and BindAllowlistAutoRun, then adapt *Runtime for the interactive UI loop.
func NewRuntime() *Runtime {
	return &Runtime{}
}

// WireSend installs send channels. Pass nil to clear.
func (r *Runtime) WireSend(s *Send) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.send = s
}

func (r *Runtime) currentSend() *Send {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.send
}

// BindAllowlistAutoRun wires the allowlist auto-run getter and sync callback.
func (r *Runtime) BindAllowlistAutoRun(getter func() bool, sync func(bool)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.allowlistFn = getter
	r.syncAllowlist = sync
}

// AllowlistAutoRunEnabled returns true when no getter is set, otherwise the getter result.
func (r *Runtime) AllowlistAutoRunEnabled() bool {
	r.mu.RLock()
	fn := r.allowlistFn
	r.mu.RUnlock()
	if fn == nil {
		return true
	}
	return fn()
}

// InvokeSyncAllowlistAutoRun runs the allowlist sync callback when non-nil.
func (r *Runtime) InvokeSyncAllowlistAutoRun(v bool) {
	r.mu.RLock()
	fn := r.syncAllowlist
	r.mu.RUnlock()
	if fn != nil {
		fn(v)
	}
}

// SetRemoteExecution updates remote execution mirror for the UI header.
func (r *Runtime) SetRemoteExecution(active bool, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remoteActive = active
	r.remoteLabel = label
}

// RemoteActive reports whether the UI should treat execution as remote.
func (r *Runtime) RemoteActive() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.remoteActive
}

// RemoteLabel returns the remote display label.
func (r *Runtime) RemoteLabel() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.remoteLabel
}

// SetOpenConfigLLMOnFirstLayout arms the next first-layout open.
func (r *Runtime) SetOpenConfigLLMOnFirstLayout(v bool) {
	r.cfgLLMMu.Lock()
	defer r.cfgLLMMu.Unlock()
	r.cfgLLMFirst = v
}

// TakeOpenConfigLLMOnFirstLayout returns whether to run startup overlay providers and clears the flag.
func (r *Runtime) TakeOpenConfigLLMOnFirstLayout() bool {
	r.cfgLLMMu.Lock()
	defer r.cfgLLMMu.Unlock()
	v := r.cfgLLMFirst
	r.cfgLLMFirst = false
	return v
}

// SubmitSubmission sends a structured submission to the host controller (blocking). Returns false if unwired.
func (r *Runtime) SubmitSubmission(sub inputlifecycletype.InputSubmission) bool {
	s := r.currentSend()
	if s == nil || s.Submission == nil {
		return false
	}
	s.Submission <- sub
	return true
}

// TrySubmitSubmissionNonBlocking sends a structured submission without blocking.
func (r *Runtime) TrySubmitSubmissionNonBlocking(sub inputlifecycletype.InputSubmission) bool {
	s := r.currentSend()
	if s == nil || s.Submission == nil {
		return false
	}
	select {
	case s.Submission <- sub:
		return true
	default:
		return false
	}
}

// NotifyConfigUpdated signals config or allowlist change (non-blocking; drops if full).
func (r *Runtime) NotifyConfigUpdated() {
	s := r.currentSend()
	if s == nil || s.ConfigUpdated == nil {
		return
	}
	select {
	case s.ConfigUpdated <- struct{}{}:
	default:
	}
}

// PublishCancelRequest forwards a cancel-processing control signal to the host controller.
func (r *Runtime) PublishCancelRequest() bool {
	s := r.currentSend()
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

// PublishShellSnapshot sends transcript lines for /sh return restore.
func (r *Runtime) PublishShellSnapshot(msgs []string) bool {
	s := r.currentSend()
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
func (r *Runtime) PublishExecDirect(cmd string) {
	if cmd == "" {
		return
	}
	s := r.currentSend()
	if s == nil || s.ExecDirect == nil {
		return
	}
	s.ExecDirect <- cmd
}

// PublishRemoteOnTarget forwards a remote connect target.
func (r *Runtime) PublishRemoteOnTarget(target string) bool {
	s := r.currentSend()
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

// PublishRemoteOff requests switching back to the local executor.
func (r *Runtime) PublishRemoteOff() bool {
	s := r.currentSend()
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

// PublishRemoteAuthResponse forwards SSH auth answers.
func (r *Runtime) PublishRemoteAuthResponse(resp remoteauth.Response) bool {
	s := r.currentSend()
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

// RequestSlashDispatch records a pre-handler slash attempt on the bus (drops if unwired or buffer full).
func (r *Runtime) RequestSlashDispatch(line string) {
	r.trySendSlashLine(line, func(s *Send) chan<- string { return s.SlashRequest })
}

// TraceSlashEntered records a dispatched slash line on the bus (drops if unwired or buffer full).
func (r *Runtime) TraceSlashEntered(line string) {
	r.trySendSlashLine(line, func(s *Send) chan<- string { return s.SlashTrace })
}

func (r *Runtime) trySendSlashLine(line string, pickCh func(*Send) chan<- string) {
	if line == "" {
		return
	}
	s := r.currentSend()
	if s == nil {
		return
	}
	ch := pickCh(s)
	if ch == nil {
		return
	}
	select {
	case ch <- line:
	default:
	}
}

// Reset clears runtime wiring and UI mirrors (for tests).
func (r *Runtime) Reset() {
	r.mu.Lock()
	r.send = nil
	r.allowlistFn = nil
	r.syncAllowlist = nil
	r.remoteActive = false
	r.remoteLabel = ""
	r.mu.Unlock()
	r.cfgLLMMu.Lock()
	r.cfgLLMFirst = false
	r.cfgLLMMu.Unlock()
}

var _ Host = (*Runtime)(nil)
