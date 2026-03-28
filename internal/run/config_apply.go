package run

import (
	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func applyConfigAllowlistUpdate(sender ui.CommandSender) inputlifecycletype.ProcessResult {
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T("en", i18n.KeyConfigPrefix) + err.Error()},
			}},
		})
	}
	if sender != nil {
		_ = sender.Send(hostcmd.ConfigUpdated{})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.Tf("en", i18n.KeyAllowlistUpdateDone, added)},
			{Kind: inputlifecycletype.TranscriptLineBlank},
		}},
	})
}
