package remote

import (
	"strings"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
)

func applyConfigRemoveRemote(nameOrTarget string) inputlifecycletype.ProcessResult {
	lang := "en"
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		return remoteTranscriptErrorResult(i18n.T(lang, i18n.KeyConfigPrefix) + "Usage: select a remote from /config del-remote list")
	}
	if err := config.RemoveRemoteByName(nameOrTarget); err != nil {
		return remoteTranscriptErrorResult(i18n.T(lang, i18n.KeyConfigPrefix) + err.Error())
	}
	return remoteTranscriptSuggestResult(i18n.Tf(lang, i18n.KeyConfigRemoteRemoved, nameOrTarget), true)
}

func remoteTranscriptSuggestResult(text string, trailingBlank bool) inputlifecycletype.ProcessResult {
	lines := []inputlifecycletype.TranscriptLine{{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: text}}
	if trailingBlank {
		lines = append(lines, inputlifecycletype.TranscriptLine{Kind: inputlifecycletype.TranscriptLineBlank})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:       inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: lines},
	})
}

func remoteTranscriptErrorResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemError, Text: text},
		}},
	})
}
