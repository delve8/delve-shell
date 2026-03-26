package run

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
)

func applyConfigAllowlistAutoRun(value string, sender ui.CommandSender) inputlifecycletype.ProcessResult {
	value = strings.TrimSpace(strings.ToLower(value))
	var on bool
	switch value {
	case "list-only":
		on = true
	case "disable":
		on = false
	default:
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T("en", i18n.KeyConfigPrefix) + i18n.T("en", i18n.KeyConfigAutoRunRequired)},
			}},
		})
	}

	cfg, err := config.Load()
	if err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T("en", i18n.KeyConfigPrefix) + err.Error()},
			}},
		})
	}
	cfg.AllowlistAutoRun = &on
	if err := config.Write(cfg); err != nil {
		return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
			Kind: inputlifecycletype.OutputTranscriptAppend,
			Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
				{Kind: inputlifecycletype.TranscriptLineSystemError, Text: i18n.T("en", i18n.KeyConfigPrefix) + err.Error()},
			}},
		})
	}
	display := i18n.T("en", i18n.KeyAutoRunListOnly)
	if !on {
		display = i18n.T("en", i18n.KeyAutoRunNone)
	}
	if sender != nil {
		_ = sender.Send(hostcmd.AllowlistAutoRun{Enabled: on})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: i18n.Tf("en", i18n.KeyConfigSavedAllowlistAutoRun, display)},
			{Kind: inputlifecycletype.TranscriptLineBlank},
		}},
	})
}

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
