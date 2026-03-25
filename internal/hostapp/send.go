package hostapp

import (
	"sync"

	"delve-shell/internal/remoteauth"
)

var (
	mu      sync.RWMutex
	installed *Send
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

// Wire installs the send bundle for this process. The last call wins. Pass nil to clear (e.g. test cleanup).
func Wire(s *Send) {
	mu.Lock()
	defer mu.Unlock()
	installed = s
}

func current() *Send {
	mu.RLock()
	defer mu.RUnlock()
	return installed
}

// Submit sends user text to the host controller (blocking). Returns false if unwired.
func Submit(text string) bool {
	s := current()
	if s == nil || s.Submit == nil {
		return false
	}
	s.Submit <- text
	return true
}

// TrySubmitNonBlocking sends without blocking; returns false if unwired or buffer full.
func TrySubmitNonBlocking(text string) bool {
	s := current()
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
	s := current()
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
	s := current()
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
	s := current()
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
	s := current()
	if s == nil || s.ExecDirect == nil {
		return
	}
	s.ExecDirect <- cmd
}

// PublishRemoteOnTarget forwards a remote connect target. Returns false if unwired or full.
func PublishRemoteOnTarget(target string) bool {
	s := current()
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
	s := current()
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
	s := current()
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
