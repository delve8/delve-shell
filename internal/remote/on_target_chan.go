package remote

import (
	"delve-shell/internal/hostapp"
	"delve-shell/internal/remoteauth"
)

// PublishRemoteOnTarget forwards a remote connect target to the host controller. Returns false if unwired or buffer full.
func PublishRemoteOnTarget(target string) bool { return hostapp.PublishRemoteOnTarget(target) }

// PublishRemoteOff requests switching back to the local executor. Returns false if unwired or buffer full.
func PublishRemoteOff() bool { return hostapp.PublishRemoteOff() }

// PublishRemoteAuthResponse forwards SSH auth answers to the host controller. Returns false if unwired or buffer full.
func PublishRemoteAuthResponse(resp remoteauth.Response) bool {
	return hostapp.PublishRemoteAuthResponse(resp)
}
