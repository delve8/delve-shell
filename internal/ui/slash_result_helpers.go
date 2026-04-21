package ui

import (
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
)

// SlashOverlayOpenResult is for slash commands that open an overlay and rely on the overlay
// itself as the immediate visible feedback beyond the echoed user command.
func SlashOverlayOpenResult(key, title, content string, markdown bool, params map[string]string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputOverlayOpen,
		Overlay: &inputlifecycletype.OverlayPayload{
			Key:      key,
			Title:    title,
			Content:  content,
			Params:   params,
			Markdown: markdown,
		},
	})
}

// SlashQuitResult is for slash commands that terminate the current TUI flow.
func SlashQuitResult() inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputQuit,
	})
}

// SlashPreInputSetResult is for slash commands that only fill the input buffer.
func SlashPreInputSetResult(value string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:     inputlifecycletype.OutputPreInputSet,
		PreInput: &inputlifecycletype.PreInputPayload{Value: value},
	})
}

// SlashProcessingResult is for slash commands that transition into an async processing flow
// (for example /skill <name> forwarding into a programmatic chat submission).
func SlashProcessingResult() inputlifecycletype.ProcessResult {
	res := inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:   inputlifecycletype.OutputStatusChange,
		Status: &inputlifecycletype.StatusPayload{Key: "processing"},
	})
	res.WaitingForAI = true
	return res
}

// SlashTranscriptSuggestResult is for slash commands that immediately append a non-error system line.
func SlashTranscriptSuggestResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: text},
		}},
	})
}

// SlashTranscriptErrorResult is for slash commands that immediately append an error line.
func SlashTranscriptErrorResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemError, Text: text},
		}},
	})
}

// SlashTryHostIntent sends a host-side intent for slash commands that rely on host/controller
// follow-up behavior (for example /access, /history, /new, /skill).
func SlashTryHostIntent(sender CommandSender, command hostcmd.Command) bool {
	return sender != nil && sender.Send(command)
}
