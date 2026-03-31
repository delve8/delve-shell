package run

import (
	"delve-shell/internal/config"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/ui"
)

func applyConfigAllowlistUpdate(sender ui.CommandSender) inputlifecycletype.ProcessResult {
	added, err := config.AllowlistUpdateWithDefaults()
	if err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T(i18n.KeyConfigPrefix) + err.Error()},
			}},
		})
	}
	if sender != nil {
		_ = sender.Send(hostcmd.ConfigUpdated{})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.Tf(i18n.KeyAllowlistUpdateDone, added)},
			{Kind: inputlifecycletype.TranscriptLineBlank},
		}},
	})
}
