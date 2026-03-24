package session

import (
	"encoding/json"
	"strings"
	"testing"

	"delve-shell/internal/history"
)

func TestSessionEventsToMessages_ConvertsEventsToDisplayLines(t *testing.T) {
	events := []history.Event{
		{Type: "user_input", Payload: json.RawMessage(`{"text":"hello"}`)},
		{Type: "llm_response", Payload: json.RawMessage(`{"reply":"hi"}`)},
		{Type: "command", Payload: json.RawMessage(`{"command":"ls","approved":true,"suggested":false}`)},
		{Type: "command_result", Payload: json.RawMessage(`{"command":"ls","stdout":"a\nb","stderr":"","exit_code":0}`)},
	}
	lines := sessionEventsToMessages(events, "en", 80)
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "User:") || !strings.Contains(lines[0], "hello") {
		t.Fatalf("unexpected first line: %q", lines[0])
	}
	if !strings.Contains(lines[2], "AI:") || !strings.Contains(lines[2], "hi") {
		t.Fatalf("unexpected AI line: %q", lines[2])
	}
	if !strings.Contains(lines[4], "ls") {
		t.Fatalf("unexpected run line: %q", lines[4])
	}
	if !strings.Contains(lines[5], "a") || !strings.Contains(lines[5], "b") {
		t.Fatalf("unexpected result line: %q", lines[5])
	}
}
