package remote

import (
	"strings"

	"delve-shell/internal/hostcmd"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config add-remote":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputOverlayOpen,
				Overlay: &inputlifecycletype.OverlayPayload{
					Key: "remote_add",
					Params: map[string]string{
						"save":    "true",
						"connect": "false",
					},
				},
			}), true, nil
		case strings.HasPrefix(text, "/config add-remote "):
			args := strings.TrimSpace(strings.TrimPrefix(text, "/config add-remote "))
			return applyConfigAddRemote(args, req.CommandSender), true, nil
		case strings.HasPrefix(text, "/config del-remote "):
			nameOrTarget := strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote "))
			return applyConfigRemoveRemote(nameOrTarget), true, nil
		case text == "/remote on":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputOverlayOpen,
				Overlay: &inputlifecycletype.OverlayPayload{
					Key: "remote_add",
					Params: map[string]string{
						"save":    "false",
						"connect": "true",
					},
				},
			}), true, nil
		case strings.HasPrefix(text, "/remote on "):
			target := strings.TrimSpace(strings.TrimPrefix(text, "/remote on "))
			if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.RemoteOnTarget{Target: target}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case text == "/remote off":
			if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.RemoteOff{}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
