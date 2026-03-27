package ui

import (
	"os"
	"path/filepath"
	"testing"

	"delve-shell/internal/uiregistry"
)

// TestGetSlashOptionsForInput_session_returnsSessionCommands asserts that
// /session suggestions are returned as command text owned by session module.
func TestGetSlashOptionsForInput_session_returnsSessionCommands(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DELVE_SHELL_ROOT", dir)

	aPath := filepath.Join(sessionsDir, "a.jsonl")
	bPath := filepath.Join(sessionsDir, "b.jsonl")
	for _, p := range []string{aPath, bPath} {
		if err := os.WriteFile(p, []byte(`{"type":"user_input","payload":{"text":"x"}}`+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	raw := uiregistry.SlashOptionsForInput("/session", "en")
	opts := make([]SlashOption, 0, len(raw))
	for _, o := range raw {
		opts = append(opts, SlashOption{Cmd: o.Cmd, Desc: o.Desc, FillValue: o.FillValue})
	}
	if len(opts) < 1 {
		t.Fatalf("expected at least 1 option, got %d", len(opts))
	}
	for _, opt := range opts {
		if len(opt.Cmd) < len("/session ") || opt.Cmd[:len("/session ")] != "/session " {
			t.Fatalf("session option must be /session <id>, got %q", opt.Cmd)
		}
	}
}
