package hostwiring

import (
	"delve-shell/internal/hostapp"
	"delve-shell/internal/hostbus"
)

// BindSendPorts wires host bus input ports and the /sh snapshot channel onto r.
func BindSendPorts(r *hostapp.Runtime, ports hostbus.InputPorts, shellSnapshot chan<- []string) {
	r.WireSend(&hostapp.Send{
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
