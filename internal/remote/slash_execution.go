package remote

import (
	"strings"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/slash/access"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case strings.HasPrefix(text, "/config remove-remote "):
			nameOrTarget := strings.TrimSpace(strings.TrimPrefix(text, "/config remove-remote "))
			return applyConfigRemoveRemote(nameOrTarget), true, nil
		case text == slashaccess.Command(slashaccess.ReservedNew):
			return ui.SlashOverlayOpenResult(OverlayOpenKeyAddRemote, "", "", false, map[string]string{
				"save": "false",
			}), true, nil
		case text == slashaccess.Command(slashaccess.ReservedLocal):
			if !ui.SlashTryHostIntent(req.CommandSender, hostcmd.AccessLocal{}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case text == slashaccess.Command(slashaccess.ReservedOffline):
			if !ui.SlashTryHostIntent(req.CommandSender, hostcmd.AccessOffline{}) {
				return inputlifecycletype.ProcessResult{}, true, nil
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case strings.HasPrefix(text, "/access "):
			accessText := text
			selectedCmd := strings.TrimSpace(req.SelectedCmd)
			selectedFill := strings.TrimSpace(req.SelectedFill)
			if selectedFill != "" && strings.HasPrefix(selectedFill, "/access ") &&
				selectedCmd != "" && strings.EqualFold(selectedCmd, text) {
				accessText = selectedFill
			}
			target := strings.TrimSpace(strings.TrimPrefix(accessText, "/access "))
			return ui.SlashOverlayOpenResult(OverlayOpenKeyAddRemote, "", "", false, map[string]string{
				"save":    "false",
				"target":  target,
				"connect": "true",
			}), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
