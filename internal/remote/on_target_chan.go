package remote

import (
	"sync"

	"delve-shell/internal/ui"
)

// Send sides for signals consumed by cli/hostloop multiplex (same channel instances as Deps).
// Set once from CLI before the UI runs.
var (
	remoteOnTargetMu sync.RWMutex
	remoteOnTargetC  chan<- string

	remoteOffMu sync.RWMutex
	remoteOffC  chan<- struct{}

	authRespMu sync.RWMutex
	authRespC  chan<- ui.RemoteAuthResponse
)

// SetRemoteOnTargetChan wires the channel used when the user confirms a remote target
// (slash / overlay). Call once per process from cli/run.go.
func SetRemoteOnTargetChan(c chan<- string) {
	remoteOnTargetMu.Lock()
	defer remoteOnTargetMu.Unlock()
	remoteOnTargetC = c
}

// trySendRemoteOnTarget sends target to the CLI multiplex. Returns false if unwired or channel full.
func trySendRemoteOnTarget(target string) bool {
	remoteOnTargetMu.RLock()
	ch := remoteOnTargetC
	remoteOnTargetMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- target:
		return true
	default:
		return false
	}
}

// SetRemoteOffChan wires the channel for /remote off (switch back to local executor).
func SetRemoteOffChan(c chan<- struct{}) {
	remoteOffMu.Lock()
	defer remoteOffMu.Unlock()
	remoteOffC = c
}

// trySendRemoteOff notifies hostloop to disconnect remote. Returns false if unwired or channel full.
func trySendRemoteOff() bool {
	remoteOffMu.RLock()
	ch := remoteOffC
	remoteOffMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// SetRemoteAuthRespChan wires SSH auth answers (password / identity path) to hostloop.
func SetRemoteAuthRespChan(c chan<- ui.RemoteAuthResponse) {
	authRespMu.Lock()
	defer authRespMu.Unlock()
	authRespC = c
}

// trySendRemoteAuthResp forwards credentials to the CLI multiplex.
func trySendRemoteAuthResp(resp ui.RemoteAuthResponse) bool {
	authRespMu.RLock()
	ch := authRespC
	authRespMu.RUnlock()
	if ch == nil {
		return false
	}
	select {
	case ch <- resp:
		return true
	default:
		return false
	}
}
