package hostwiring

import (
	"delve-shell/internal/hostapp"
	"delve-shell/internal/hostbus"
)

// BindSendPorts installs a single hostapp.Send bundle from host bus input ports plus the /sh snapshot channel.
func BindSendPorts(ports hostbus.InputPorts, shellSnapshot chan<- []string) {
	hostapp.Wire(&hostapp.Send{
		Submit:         ports.SubmitChan,
		ConfigUpdated:  ports.ConfigUpdatedChan,
		CancelRequest:  ports.CancelRequestChan,
		ExecDirect:     ports.ExecDirectChan,
		RemoteOn:       ports.RemoteOnChan,
		RemoteOff:      ports.RemoteOffChan,
		RemoteAuthResp: ports.RemoteAuthRespChan,
		ShellSnapshot:  shellSnapshot,
	})
}
