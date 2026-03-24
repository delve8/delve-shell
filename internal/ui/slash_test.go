package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGetSlashOptionsForInput_sessions_returnsSessionCommands asserts that
// /sessions suggestions are returned as command text owned by session module.
func TestGetSlashOptionsForInput_sessions_returnsSessionCommands(t *testing.T) {
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

	opts := getSlashOptionsForInput("/sessions", "en", nil, nil, false)
	if len(opts) < 1 {
		t.Fatalf("expected at least 1 option, got %d", len(opts))
	}
	for _, opt := range opts {
		if len(opt.Cmd) < len("/sessions ") || opt.Cmd[:len("/sessions ")] != "/sessions " {
			t.Fatalf("session option must be /sessions <id>, got %q", opt.Cmd)
		}
	}
}

func TestGetSlashOptionsForInput_runCompletion_filtersAndNoFallback(t *testing.T) {
	local := []string{"bash", "base64", "cat"}

	// "/run" shows the usage row.
	opts := getSlashOptionsForInput("/run", "en", local, nil, false)
	if len(opts) != 1 || opts[0].Cmd != SlashRunUsageOption {
		t.Fatalf("expected usage option for /run, got %#v", opts)
	}

	// "/run b" filters local commands.
	opts = getSlashOptionsForInput("/run b", "en", local, nil, false)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d: %#v", len(opts), opts)
	}
	if opts[0].Cmd != "/run bash" || opts[1].Cmd != "/run base64" {
		t.Fatalf("unexpected options: %#v", opts)
	}

	// No match: return empty (do not fall back to top-level slash commands).
	opts = getSlashOptionsForInput("/run z", "en", local, nil, false)
	if len(opts) != 0 {
		t.Fatalf("expected no options for unmatched /run prefix, got %#v", opts)
	}
}

func TestGetSlashOptionsForInput_runCompletion_usesRemoteCacheWhenActive(t *testing.T) {
	local := []string{"bash", "base64"}
	remote := []string{"busybox", "bzip2"}

	opts := getSlashOptionsForInput("/run b", "en", local, remote, true)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options from remote cache, got %d: %#v", len(opts), opts)
	}
	if opts[0].Cmd != "/run busybox" || opts[1].Cmd != "/run bzip2" {
		t.Fatalf("unexpected options: %#v", opts)
	}
}
