package run

import (
	"strings"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		if text == "/config update auto-run list" {
			return applyConfigAllowlistUpdate(req.CommandSender), true, nil
		}
		return inputlifecycletype.ProcessResult{}, false, nil
	})
}
