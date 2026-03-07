package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListSessions_emptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	// history dir does not exist yet
	paths, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(paths) != 0 {
		t.Errorf("expected no paths in empty root, got %d", len(paths))
	}
}

func TestListSessions_returnsOnlyJsonl(t *testing.T) {
	dir := t.TempDir()
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DELVE_SHELL_ROOT", dir)

	// create one .jsonl and one non-jsonl file
	if err := os.WriteFile(filepath.Join(historyDir, "a.jsonl"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(historyDir, "b.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	paths, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(paths) != 1 {
		t.Errorf("expected 1 path (.jsonl only), got %d", len(paths))
	}
	if len(paths) > 0 && filepath.Base(paths[0]) != "a.jsonl" {
		t.Errorf("expected a.jsonl, got %s", paths[0])
	}
}

func TestReadRecent_maxLinesKeepsLast(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "s.jsonl")
	f, err := os.Create(fpath)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for i := 0; i < 10; i++ {
		ev := Event{Type: "user_input", Payload: json.RawMessage(`{"text":"` + string(rune('0'+i)) + `"}`)}
		if err := enc.Encode(ev); err != nil {
			f.Close()
			t.Fatal(err)
		}
	}
	f.Close()

	events, err := ReadRecent(fpath, 3)
	if err != nil {
		t.Fatalf("ReadRecent: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events (last 3), got %d", len(events))
	}
	// last 3 should be 7, 8, 9
	var last struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(events[2].Payload, &last); err != nil || last.Text != "9" {
		t.Errorf("expected last payload text 9, got %q err=%v", last.Text, err)
	}
}

func TestReadRecent_noFileReturnsNilNil(t *testing.T) {
	events, err := ReadRecent(filepath.Join(t.TempDir(), "nonexistent.jsonl"), 10)
	if err != nil {
		t.Errorf("expected nil error for missing file, got %v", err)
	}
	if events != nil {
		t.Errorf("expected nil events for missing file, got %d", len(events))
	}
}

func TestListSessionsWithSummary_sortedByMtimeDesc(t *testing.T) {
	dir := t.TempDir()
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DELVE_SHELL_ROOT", dir)

	// create two session files with different mtimes
	p1 := filepath.Join(historyDir, "old.jsonl")
	p2 := filepath.Join(historyDir, "new.jsonl")
	if err := os.WriteFile(p1, []byte(`{"type":"user_input","payload":{"text":"old"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(p2, []byte(`{"type":"user_input","payload":{"text":"new"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	summaries, err := ListSessionsWithSummary(10)
	if err != nil {
		t.Fatalf("ListSessionsWithSummary: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}
	// newest first
	if summaries[0].ID != "new" {
		t.Errorf("expected first (newest) ID new, got %s", summaries[0].ID)
	}
	if summaries[1].ID != "old" {
		t.Errorf("expected second ID old, got %s", summaries[1].ID)
	}
}
