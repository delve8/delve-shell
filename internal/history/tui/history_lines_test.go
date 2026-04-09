package historytui

import (
	"encoding/json"
	"strings"
	"testing"

	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

func TestEventsToTranscriptLinesForHistoryPreview_fullCommandText(t *testing.T) {
	longCmd := strings.Repeat("c", 200)
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"` + longCmd + `","approved":true,"suggested":false}`)},
	}
	lines := EventsToTranscriptLinesForHistoryPreview(events)
	if len(lines) < 1 {
		t.Fatal("expected exec line")
	}
	if lines[0].Kind != uivm.LineExec {
		t.Fatalf("want LineExec, got %v", lines[0].Kind)
	}
	if len(lines[0].Text) < len(longCmd) {
		t.Fatalf("command truncated in preview line: len %d", len(lines[0].Text))
	}
}

func TestEventsToTranscriptLines_ConvertsEventsToSemanticLines(t *testing.T) {
	i18n.SetLang("en")
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

func TestEventsToTranscriptLines_OfflineManualUsesManualPrefix(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl get pods","approved":true,"execution":"offline_manual","offline_mode":true}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 1 {
		t.Fatal("expected exec line")
	}
	if lines[0].Text != "Run (manual): kubectl get pods" {
		t.Fatalf("unexpected manual prefix: %q", lines[0].Text)
	}
}

func TestEventsToTranscriptLinesForHistoryPreview_OfflineManualUsesManualPrefix(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl get pods","approved":true,"execution":"offline_manual","offline_mode":true}`)},
	}
	lines := EventsToTranscriptLinesForHistoryPreview(events)
	if len(lines) < 1 {
		t.Fatal("expected exec line")
	}
	if lines[0].Text != "Run (manual): kubectl get pods" {
		t.Fatalf("unexpected manual preview prefix: %q", lines[0].Text)
	}
}
