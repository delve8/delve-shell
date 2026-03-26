package wiring

import (
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
)

// BindSendPorts wires host bus input ports and the /sh snapshot channel onto r.
func BindSendPorts(r *app.Runtime, ports bus.InputPorts, shellSnapshot chan<- []string) {
	r.WireSend(&app.Send{
		Submission:     ports.SubmissionChan,
		ConfigUpdated:  ports.ConfigUpdatedChan,
		CancelRequest:  ports.CancelRequestChan,
		ExecDirect:     ports.ExecDirectChan,
		RemoteOn:       ports.RemoteOnChan,
		RemoteOff:      ports.RemoteOffChan,
		RemoteAuthResp: ports.RemoteAuthRespChan,
		ShellSnapshot:  shellSnapshot,
	})
}
