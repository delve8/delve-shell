package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadRecent_LongLineDoesNotBreak(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")

	long := strings.Repeat("x", 200_000)
	ev := Event{
		Time:    time.Now().UTC(),
		Type:    "llm_response",
		Payload: json.RawMessage(`{"reply":"` + long + `"}`),
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(p, append(b, '\n'), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, err := ReadRecent(p, 50)
	if err != nil {
		t.Fatalf("ReadRecent err: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 event, got %d", len(out))
	}
	if out[0].Type != "llm_response" {
		t.Fatalf("unexpected type: %q", out[0].Type)
	}
}

func TestReadRecent_MaxLinesKeepsTail(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")

	var sb strings.Builder
	for i := 0; i < 5; i++ {
		ev := Event{Time: time.Now().UTC(), Type: "user_input", Payload: json.RawMessage(`{"text":"` + string(rune('a'+i)) + `"}`)}
		b, err := json.Marshal(ev)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(p, []byte(sb.String()), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out, err := ReadRecent(p, 2)
	if err != nil {
		t.Fatalf("ReadRecent err: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 events, got %d", len(out))
	}
	// Expect last two payloads: d,e
	if !strings.Contains(string(out[0].Payload), `"text":"d"`) {
		t.Fatalf("unexpected payload[0]: %s", string(out[0].Payload))
	}
	if !strings.Contains(string(out[1].Payload), `"text":"e"`) {
		t.Fatalf("unexpected payload[1]: %s", string(out[1].Payload))
	}
}

