package run

import (
	"delve-shell/internal/inputlifecycletype"
)

func transcriptSuggestResult(text string, trailingBlank bool) inputlifecycletype.ProcessResult {
	lines := []inputlifecycletype.TranscriptLine{{Kind: inputlifecycletype.TranscriptLineSystemSuggest, Text: text}}
	if trailingBlank {
		lines = append(lines, inputlifecycletype.TranscriptLine{Kind: inputlifecycletype.TranscriptLineBlank})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:       inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: lines},
	})
}

func transcriptErrorResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputTranscriptAppend,
		Transcript: &inputlifecycletype.TranscriptPayload{Lines: []inputlifecycletype.TranscriptLine{
			{Kind: inputlifecycletype.TranscriptLineSystemError, Text: text},
		}},
	})
}
