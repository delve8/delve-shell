package run

import (
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func transcriptSuggestResult(text string, trailingBlank bool) inputlifecycletype.ProcessResult {
	lines := []uivm.Line{{Kind: uivm.LineSystemSuggest, Text: text}}
	if trailingBlank {
		lines = append(lines, uivm.Line{Kind: uivm.LineBlank})
	}
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{
			Value: ui.TranscriptAppendMsg{Lines: lines},
		},
	})
}

func transcriptErrorResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{
			Value: ui.TranscriptAppendMsg{Lines: []uivm.Line{
				{Kind: uivm.LineSystemError, Text: text},
			}},
		},
	})
}
