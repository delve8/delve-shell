package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/service/remotesvc"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func applyConfigAddRemote(args string, sender ui.ActionSender) inputlifecycletype.ProcessResult {
	lang := "en"
	parts := strings.Fields(args)
	if len(parts) < 1 {
		return remoteTranscriptErrorResult(i18n.T(lang, i18n.KeyConfigPrefix) + "Usage: /config add-remote <user@host> [name] [identity_file]")
	}
	target := parts[0]
	name := ""
	identityFile := ""
	if len(parts) >= 2 {
		name = parts[1]
	}
	if len(parts) >= 3 {
		identityFile = parts[2]
	}
	if err := remotesvc.Add(target, name, identityFile); err != nil {
		return remoteTranscriptErrorResult(i18n.T(lang, i18n.KeyConfigPrefix) + err.Error())
	}
	display := target
	if name != "" {
		display = name + " (" + target + ")"
	}
	if sender != nil {
		_ = sender.Send(uivm.UIAction{Kind: uivm.UIActionConfigUpdated})
	}
	return remoteTranscriptSuggestResult(i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display), true)
}

func applyConfigRemoveRemote(nameOrTarget string) inputlifecycletype.ProcessResult {
	lang := "en"
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		return remoteTranscriptErrorResult(i18n.T(lang, i18n.KeyConfigPrefix) + "Usage: select a remote from /config del-remote list")
	}
	if err := remotesvc.Remove(nameOrTarget); err != nil {
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
