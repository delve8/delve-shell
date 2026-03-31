package historytui

import (
	"encoding/json"
	"testing"

	"delve-shell/internal/history"
)

func TestEventsToTranscriptLines_ConvertsEventsToSemanticLines(t *testing.T) {
	events := []history.Event{
		{Type: history.EventTypeUserInput, Payload: json.RawMessage(`{"text":"hello"}`)},
		{Type: history.EventTypeLLMResponse, Payload: json.RawMessage(`{"reply":"hi"}`)},
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"ls","approved":true,"suggested":false}`)},
		{Type: history.EventTypeCommandResult, Payload: json.RawMessage(`{"command":"ls","stdout":"a\nb","stderr":"","exit_code":0}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 semantic lines, got %d", len(lines))
	}
}
