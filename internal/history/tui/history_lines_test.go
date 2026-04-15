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

func TestEventsToTranscriptLines_multilineCommandCompacted(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl get nodes \\\n  -o wide\nkubectl get pods -A","suggested":false}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 1 {
		t.Fatal("expected exec line")
	}
	want := "Run (approved): kubectl get nodes \\ -o wide kubectl get pods -A"
	if lines[0].Text != want {
		t.Fatalf("got %q want %q", lines[0].Text, want)
	}
}

func TestEventsToTranscriptLinesForHistoryPreview_multilineCommandPreserved(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl get nodes \\\n  -o wide\nkubectl get pods -A","suggested":false}`)},
	}
	lines := EventsToTranscriptLinesForHistoryPreview(events)
	if len(lines) < 1 {
		t.Fatal("expected exec line")
	}
	indent := strings.Repeat(" ", len("Run (approved): "))
	want := "Run (approved): kubectl get nodes \\\n" + indent + "  -o wide\n" + indent + "kubectl get pods -A"
	if lines[0].Text != want {
		t.Fatalf("got %q want %q", lines[0].Text, want)
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
	if lines[0].Text != "Run (manual) @ Offline: kubectl get pods" {
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
	if lines[0].Text != "Run (manual) @ Offline: kubectl get pods" {
		t.Fatalf("unexpected manual preview prefix: %q", lines[0].Text)
	}
}

func TestEventsToTranscriptLines_IncludesExecutionTarget(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"pwd","approved":true,"execution":"local","execution_target":"Local"}`)},
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"hostname","approved":true,"execution":"remote","execution_target":"prod (10.0.0.1)"}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 2 {
		t.Fatalf("expected exec lines, got %d", len(lines))
	}
	if lines[0].Text != "Run (approved) @ Local: pwd" {
		t.Fatalf("unexpected local line: %q", lines[0].Text)
	}
	if lines[1].Text != "Run (approved) @ Remote prod (10.0.0.1): hostname" {
		t.Fatalf("unexpected remote line: %q", lines[1].Text)
	}
}

func TestEventsToTranscriptLines_AutoAllowedUsesChecksPassedPrefix(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl get pods","approved":true,"auto_allowed":true,"execution":"local","execution_target":"Local"}`)},
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl delete pod x","approved":true,"execution":"local","execution_target":"Local"}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 2 {
		t.Fatalf("expected exec lines, got %d", len(lines))
	}
	if lines[0].Text != "Run (checks passed) @ Local: kubectl get pods" {
		t.Fatalf("unexpected auto-allowed line: %q", lines[0].Text)
	}
	if lines[1].Text != "Run (approved) @ Local: kubectl delete pod x" {
		t.Fatalf("unexpected approved line: %q", lines[1].Text)
	}
}

func TestEventsToTranscriptLines_NotApprovedUsesNotApprovedPrefix(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl delete pod x","approved":false,"execution":"local","execution_target":"Local"}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 1 {
		t.Fatalf("expected exec line, got %d", len(lines))
	}
	if lines[0].Text != "Run (not approved) @ Local: kubectl delete pod x" {
		t.Fatalf("unexpected not-approved line: %q", lines[0].Text)
	}
}

func TestEventsToTranscriptLines_IncludesGuidanceLine(t *testing.T) {
	i18n.SetLang("en")
	events := []history.Event{
		{Type: history.EventTypeCommand, Payload: json.RawMessage(`{"command":"kubectl delete pod x","approved":false,"guidance":"check logs first","execution":"local","execution_target":"Local"}`)},
	}
	lines := EventsToTranscriptLines(events)
	if len(lines) < 2 {
		t.Fatalf("expected exec and guidance lines, got %d", len(lines))
	}
	if lines[0].Text != "Run (not approved) @ Local: kubectl delete pod x" {
		t.Fatalf("unexpected command line: %q", lines[0].Text)
	}
	if lines[1].Text != "User guidance: check logs first" {
		t.Fatalf("unexpected guidance line: %q", lines[1].Text)
	}
}
