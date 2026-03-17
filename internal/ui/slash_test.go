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

func TestGetSlashOptionsForInput_runCompletion_filtersAndNoFallback(t *testing.T) {
	local := []string{"bash", "base64", "cat"}

	// "/run" shows the usage row.
	opts := getSlashOptionsForInput("/run", "en", "", local, nil, false)
	if len(opts) != 1 || opts[0].Cmd != "/run <cmd>" {
		t.Fatalf("expected usage option for /run, got %#v", opts)
	}

	// "/run b" filters local commands.
	opts = getSlashOptionsForInput("/run b", "en", "", local, nil, false)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d: %#v", len(opts), opts)
	}
	if opts[0].Cmd != "/run bash" || opts[1].Cmd != "/run base64" {
		t.Fatalf("unexpected options: %#v", opts)
	}

	// No match: return empty (do not fall back to top-level slash commands).
	opts = getSlashOptionsForInput("/run z", "en", "", local, nil, false)
	if len(opts) != 0 {
		t.Fatalf("expected no options for unmatched /run prefix, got %#v", opts)
	}
}

func TestGetSlashOptionsForInput_runCompletion_usesRemoteCacheWhenActive(t *testing.T) {
	local := []string{"bash", "base64"}
	remote := []string{"busybox", "bzip2"}

	opts := getSlashOptionsForInput("/run b", "en", "", local, remote, true)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options from remote cache, got %d: %#v", len(opts), opts)
	}
	if opts[0].Cmd != "/run busybox" || opts[1].Cmd != "/run bzip2" {
		t.Fatalf("unexpected options: %#v", opts)
	}
}
