package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
)

func TestBlackboxSlashHelpOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/help")
	if !got.Overlay.Active {
		t.Fatalf("expected /help to open overlay")
	}
	if got.Overlay.Title == "" {
		t.Fatalf("expected /help overlay title to be non-empty")
	}
}

func TestBlackboxSlashShSendsMessagesToShell(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model = f.model.WithTranscriptLines([]string{"a", "b"})

	_ = enterText(f.model, "/sh")
	select {
	case snap := <-f.shellRequested:
		msgs := snap.Messages
		if len(msgs) < 2 || msgs[0] != "a" || msgs[1] != "b" {
			t.Fatalf("unexpected shell message snapshot prefix: %#v", msgs)
		}
		if snap.Mode != hostcmd.SubshellModeLocalBash {
			t.Fatalf("expected local bash subshell mode, got %v", snap.Mode)
		}
		joined := strings.Join(msgs, "\n")
		if !strings.Contains(joined, "User: /sh") {
			t.Fatalf("expected User echo for /sh in snapshot, got %#v", msgs)
		}
	default:
		t.Fatalf("expected /sh to send message snapshot")
	}
}

func TestBlackboxSlashShRemoteModeWhenRemoteActive(t *testing.T) {
	f := newBlackboxFixture(t)
	next, _ := f.model.Update(remote.ExecutionChangedMsg{Active: true, Label: "r1"})
	m := next.(ui.Model)
	_ = enterText(m, "/sh")
	select {
	case snap := <-f.shellRequested:
		if snap.Mode != hostcmd.SubshellModeRemoteSSH {
			t.Fatalf("expected remote SSH subshell mode, got %v", snap.Mode)
		}
	default:
		t.Fatalf("expected /sh snapshot")
	}
}

func TestBlackboxSlashRunExecutesDirectCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	_ = enterText(f.model, "/run echo")
	select {
	case cmd := <-f.execDirectChan:
		if cmd != "echo" {
			t.Fatalf("expected exec cmd 'echo', got %q", cmd)
		}
	default:
		t.Fatalf("expected /run to send command to execDirectChan")
	}
}

func TestBlackboxSlashRunUsageFillsInput(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/run")
	if got.Input.Value() != "/run " {
		t.Fatalf("expected /run to fill input to '/run ', got %q", got.Input.Value())
	}
}

func TestBlackboxSlashRunDropdownUsesRemoteCachedSuggestionsWhenAvailable(t *testing.T) {
	f := newBlackboxFixture(t)

	next, _ := f.model.Update(remote.ExecutionChangedMsg{Active: true, Label: "r1"})
	m1 := next.(ui.Model)
	next2, _ := m1.Update(remote.RunCompletionCacheMsg{RemoteLabel: "r1", Commands: []string{"busybox", "bzip2"}})
	m2 := next2.(ui.Model)

	m2.Input.SetValue("/run b")
	m2.Input.CursorEnd()
	view := m2.View()
	if !strings.Contains(view, "/run busybox") || !strings.Contains(view, "/run bzip2") {
		t.Fatalf("expected remote cached /run suggestions in dropdown, got view:\n%s", view)
	}
}

func TestBlackboxSlashConfigDelRemoteNoHostsShowsHint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config del-remote")
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after no-hosts del-remote, got %q", got.Input.Value())
	}
	joined := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(joined, "No hosts") {
		t.Fatalf("expected no-hosts hint in transcript, got %q", joined)
	}
}

func TestBlackboxSlashConfigFillsToFirstSubcommandOnEnter(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config")
	if got.Input.Value() != "/config del-remote" {
		t.Fatalf("expected /config to fill to first subcommand, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashDropdownUpDownAndEnterFill(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := next.(ui.Model)
	if m2.Input.Value() != "/" {
		t.Fatalf("expected input to remain '/', got %q", m2.Input.Value())
	}

	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(ui.Model)
	if strings.TrimSpace(m3.Input.Value()) == "/" {
		t.Fatalf("expected enter to fill a concrete slash option, got %q", m3.Input.Value())
	}
	if v := strings.TrimSpace(m3.Input.Value()); v != "" && !strings.HasPrefix(v, "/") {
		t.Fatalf("expected filled value to start with '/', got %q", m3.Input.Value())
	}
}
