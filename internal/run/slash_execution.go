package run

import (
	"strings"

	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config show":
			return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
				Kind: inputlifecycletype.OutputTranscriptAppend,
				Transcript: &inputlifecycletype.TranscriptPayload{
					Lines: []inputlifecycletype.TranscriptLine{
						{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.T("en", i18n.KeyConfigHint)},
					},
				},
			}), true, nil
		case text == "/config update auto-run list":
			return applyConfigAllowlistUpdate(req.CommandSender), true, nil
		case text == "/config reload", text == "/reload":
			if req.CommandSender != nil {
				_ = req.CommandSender.Send(hostcmd.ConfigUpdated{})
			}
			return inputlifecycletype.ConsumedResult(), true, nil
		case strings.HasPrefix(text, "/config auto-run "):
			return applyConfigAllowlistAutoRun(strings.TrimSpace(strings.TrimPrefix(text, "/config auto-run ")), req.CommandSender), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}
