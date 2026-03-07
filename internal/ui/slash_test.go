package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetSessionSlashOptions_excludesCurrentSession asserts that the session list does not include currentSessionPath.
func TestGetSessionSlashOptions_excludesCurrentSession(t *testing.T) {
	dir := t.TempDir()
	historyDir := filepath.Join(dir, "history")
	if err := os.MkdirAll(historyDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DELVE_SHELL_ROOT", dir)

	aPath := filepath.Join(historyDir, "a.jsonl")
	bPath := filepath.Join(historyDir, "b.jsonl")
	for _, p := range []string{aPath, bPath} {
		if err := os.WriteFile(p, []byte(`{"type":"user_input","payload":{"text":"x"}}`+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	opts := getSessionSlashOptions("", aPath)
	if len(opts) != 1 {
		t.Fatalf("expected 1 option (current a excluded), got %d", len(opts))
	}
	if opts[0].Path == aPath {
		t.Error("current session path should be excluded from list")
	}
	if opts[0].Path != bPath {
		t.Errorf("expected remaining option to be b, got %s", opts[0].Path)
	}
	if !strings.HasPrefix(opts[0].Cmd, "b") {
		t.Errorf("option Cmd should start with session id b, got %q", opts[0].Cmd)
	}
}
