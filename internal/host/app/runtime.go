package app

import (
	"fmt"
	"strings"
	"sync"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/remote/auth"
)

// Runtime holds host wiring and UI mirrors for one process (or one test fixture).
type Runtime struct {
	mu sync.RWMutex
	// send is the channel bundle installed by WireSend (nil when unwired).
	send          *Send
	remoteActive  bool
	remoteLabel   string // display string for UI (e.g. "name (host)" or host only)
	remoteHost    string // hostname or IP from SSH target (no "name ( )" wrapper)
	remoteName    string // remotes.yaml entry name when configured; may be empty
	offline       bool
	cfgModelMu    sync.Mutex
	cfgModelFirst bool
}

// NewRuntime returns an empty runtime; call WireSend, then adapt *Runtime for the interactive UI loop.
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

// SetRemoteExecution updates remote execution mirror for the UI footer/status bar and LLM exec context.
// When active: label is the display string (same as TUI); host is hostname or IP from the SSH target; configName is the remotes.yaml profile name (may be empty).
func (r *Runtime) SetRemoteExecution(active bool, label, host, configName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.remoteActive = active
	if active {
		r.remoteLabel = strings.TrimSpace(label)
		r.remoteHost = strings.TrimSpace(host)
		r.remoteName = strings.TrimSpace(configName)
		r.offline = false
		return
	}
	r.remoteLabel = ""
	r.remoteHost = ""
	r.remoteName = ""
}

// SetOffline sets offline (manual relay) mode mirror for the UI; clears remote active.
func (r *Runtime) SetOffline(v bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.offline = v
	if v {
		r.remoteActive = false
		r.remoteLabel = ""
		r.remoteHost = ""
		r.remoteName = ""
	}
}

// Offline reports whether offline execution mode is active.
func (r *Runtime) Offline() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.offline
}

// OfflineExecutionMode implements Host.
func (r *Runtime) OfflineExecutionMode() bool {
	return r.Offline()
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

// ParseRemoteDisplayLabel splits labels built as "name (host)" for remotes.yaml entries.
// If the pattern does not match, name is empty and host is the trimmed whole string.
func ParseRemoteDisplayLabel(display string) (name, host string) {
	s := strings.TrimSpace(display)
	if s == "" {
		return "", ""
	}
	if i := strings.LastIndex(s, " ("); i > 0 && strings.HasSuffix(s, ")") {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+2 : len(s)-1])
	}
	return "", s
}

// ExecContextForLLM returns a short English line for the LLM: current execution node only (local / remote / offline).
func (r *Runtime) ExecContextForLLM() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.offline {
		return "Offline (manual relay)"
	}
	if !r.remoteActive {
		return "Local"
	}
	host := r.remoteHost
	name := r.remoteName
	if host == "" && name == "" {
		var parsedHost string
		name, parsedHost = ParseRemoteDisplayLabel(r.remoteLabel)
		host = parsedHost
	}
	if host == "" {
		host = strings.TrimSpace(r.remoteLabel)
	}
	if name != "" && host != "" {
		return fmt.Sprintf("Remote: %s @ %s", name, host)
	}
	if host != "" {
		return fmt.Sprintf("Remote: %s", host)
	}
	return "Remote"
}

// SetOpenConfigModelOnFirstLayout arms the next first-layout open.
func (r *Runtime) SetOpenConfigModelOnFirstLayout(v bool) {
	r.cfgModelMu.Lock()
	defer r.cfgModelMu.Unlock()
	r.cfgModelFirst = v
}

// TakeOpenConfigModelOnFirstLayout returns whether to run startup overlay providers and clears the flag.
func (r *Runtime) TakeOpenConfigModelOnFirstLayout() bool {
	r.cfgModelMu.Lock()
	defer r.cfgModelMu.Unlock()
	v := r.cfgModelFirst
	r.cfgModelFirst = false
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

// PublishShellSnapshot sends transcript lines for /bash return restore.
func (r *Runtime) PublishShellSnapshot(snap hostcmd.ShellSnapshot) bool {
	s := r.currentSend()
	if s == nil || s.ShellSnapshot == nil {
		return false
	}
	select {
	case s.ShellSnapshot <- snap:
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

// PublishAccessRemote forwards a remote connect target.
func (r *Runtime) PublishAccessRemote(target string) bool {
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

// PublishAccessLocal requests switching back to the local executor.
func (r *Runtime) PublishAccessLocal() bool {
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

// Reset clears runtime wiring and UI mirrors (for tests).
func (r *Runtime) Reset() {
	r.mu.Lock()
	r.send = nil
	r.remoteActive = false
	r.remoteLabel = ""
	r.remoteHost = ""
	r.remoteName = ""
	r.offline = false
	r.mu.Unlock()
	r.cfgModelMu.Lock()
	r.cfgModelFirst = false
	r.cfgModelMu.Unlock()
}

var _ Host = (*Runtime)(nil)
