package remote

import (
	"strings"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case strings.HasPrefix(text, "/config del-remote "):
			nameOrTarget := strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote "))
			return applyConfigRemoveRemote(nameOrTarget), true, nil
		case text == "/access New":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputOverlayOpen,
				Overlay: &inputlifecycletype.OverlayPayload{
					Key: "remote_add",
					Params: map[string]string{
						"save": "false",
					},
				},
			}), true, nil
		case text == "/access Local":
			if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.RemoteOff{}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case text == "/access Offline":
			if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.AccessOffline{}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case strings.HasPrefix(text, "/access "):
			target := strings.TrimSpace(strings.TrimPrefix(text, "/access "))
			if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.RemoteOnTarget{Target: target}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
