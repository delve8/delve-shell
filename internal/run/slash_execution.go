package run

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config show":
			return transcriptSuggestResult(i18n.T("en", i18n.KeyConfigHint), false), true, nil
		case text == "/config update auto-run list":
			return applyConfigAllowlistUpdate(req.ActionSender), true, nil
		case text == "/config reload", text == "/reload":
			if req.ActionSender != nil {
				_ = req.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionConfigUpdated})
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case strings.HasPrefix(text, "/config auto-run "):
			return applyConfigAllowlistAutoRun(strings.TrimSpace(strings.TrimPrefix(text, "/config auto-run ")), req.ActionSender), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
