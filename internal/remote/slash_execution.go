package remote

import (
	"strings"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config add-remote":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputMessage,
				Message: &inputlifecycletype.MessagePayload{Value: OpenAddRemoteOverlayMsg{
					Save:    true,
					Connect: false,
				}},
			}), true, nil
		case strings.HasPrefix(text, "/config add-remote "):
			args := strings.TrimSpace(strings.TrimPrefix(text, "/config add-remote "))
			return applyConfigAddRemote(args, req.ActionSender), true, nil
		case strings.HasPrefix(text, "/config del-remote "):
			nameOrTarget := strings.TrimSpace(strings.TrimPrefix(text, "/config del-remote "))
			return applyConfigRemoveRemote(nameOrTarget), true, nil
		case text == "/remote on":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputMessage,
				Message: &inputlifecycletype.MessagePayload{Value: OpenAddRemoteOverlayMsg{
					Save:    false,
					Connect: true,
				}},
			}), true, nil
		case strings.HasPrefix(text, "/remote on "):
			target := strings.TrimSpace(strings.TrimPrefix(text, "/remote on "))
			if req.ActionSender == nil || !req.ActionSender.Send(uivm.UIAction{
				Kind: uivm.UIActionRemoteOnTarget,
				Text: target,
			}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case text == "/remote off":
			if req.ActionSender == nil || !req.ActionSender.Send(uivm.UIAction{
				Kind: uivm.UIActionRemoteOff,
			}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
