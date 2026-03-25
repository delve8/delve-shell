package interactive

import (
	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/remote"
	"delve-shell/internal/run"
)

// WireHostChannels connects hostbus input ports and auxiliary channels to legacy package-level setters.
// Feature packages (run, remote, hostnotify) still publish through these globals until a later migration.
func WireHostChannels(ports hostbus.InputPorts, shellRequestedChan chan<- []string) {
	hostnotify.SetSubmitChan(ports.SubmitChan)
	run.SetExecDirectChan(ports.ExecDirectChan)
	hostnotify.SetConfigUpdatedChan(ports.ConfigUpdatedChan)
	run.SetShellRequestedChan(shellRequestedChan)
	run.SetCancelRequestChan(ports.CancelRequestChan)
	remote.SetRemoteOnTargetChan(ports.RemoteOnChan)
	remote.SetRemoteOffChan(ports.RemoteOffChan)
	remote.SetRemoteAuthRespChan(ports.RemoteAuthRespChan)
}
