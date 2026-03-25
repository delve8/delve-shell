package hostwiring

import (
	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/remote"
	"delve-shell/internal/run"
)

// BindLegacyFeatureChannels wires hostbus input ports and the /sh return snapshot channel to package-level setters.
// The last call wins for each global; tests must not run in parallel with other packages that re-bind the same globals.
func BindLegacyFeatureChannels(ports hostbus.InputPorts, shellRequestedChan chan<- []string) {
	hostnotify.SetSubmitChan(ports.SubmitChan)
	run.SetExecDirectChan(ports.ExecDirectChan)
	hostnotify.SetConfigUpdatedChan(ports.ConfigUpdatedChan)
	run.SetShellRequestedChan(shellRequestedChan)
	run.SetCancelRequestChan(ports.CancelRequestChan)
	remote.SetRemoteOnTargetChan(ports.RemoteOnChan)
	remote.SetRemoteOffChan(ports.RemoteOffChan)
	remote.SetRemoteAuthRespChan(ports.RemoteAuthRespChan)
}
