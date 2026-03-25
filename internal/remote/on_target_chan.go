package remote

import (
	"sync"

	"delve-shell/internal/remoteauth"
)

// Send sides for signals consumed by the host controller via hostbus input ports.
// Set once from CLI before the UI runs.
var (
	remoteOnTargetMu sync.RWMutex
	remoteOnTargetC  chan<- string

	remoteOffMu sync.RWMutex
	remoteOffC  chan<- struct{}

	authRespMu sync.RWMutex
	authRespC  chan<- remoteauth.Response
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

// trySendRemoteOff notifies the host controller to disconnect remote. Returns false if unwired or channel full.
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

// SetRemoteAuthRespChan wires SSH auth answers (password / identity path) to the host controller.
func SetRemoteAuthRespChan(c chan<- remoteauth.Response) {
	authRespMu.Lock()
	defer authRespMu.Unlock()
	authRespC = c
}

// trySendRemoteAuthResp forwards credentials to the CLI multiplex.
func trySendRemoteAuthResp(resp remoteauth.Response) bool {
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

// PublishRemoteOnTarget forwards a remote connect target to the host controller. Returns false if unwired or buffer full.
func PublishRemoteOnTarget(target string) bool { return trySendRemoteOnTarget(target) }

// PublishRemoteOff requests switching back to the local executor. Returns false if unwired or buffer full.
func PublishRemoteOff() bool { return trySendRemoteOff() }

// PublishRemoteAuthResponse forwards SSH auth answers to the host controller. Returns false if unwired or buffer full.
func PublishRemoteAuthResponse(resp remoteauth.Response) bool { return trySendRemoteAuthResp(resp) }
