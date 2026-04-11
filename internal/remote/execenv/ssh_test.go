package execenv

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestParseUserHost_RequiresExplicitUsername(t *testing.T) {
	_, _, err := parseUserHost("example.com")
	if err == nil {
		t.Fatal("expected missing username to fail")
	}
	if got, want := err.Error(), "ssh target must include username (user@host or user@host:port)"; got != want {
		t.Fatalf("error=%q want %q", got, want)
	}
}

func TestParseUserHost_AddsDefaultPort(t *testing.T) {
	user, hostPort, err := parseUserHost("alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "alice" {
		t.Fatalf("user=%q want %q", user, "alice")
	}
	if hostPort != "example.com:22" {
		t.Fatalf("hostPort=%q want %q", hostPort, "example.com:22")
	}
}

func TestSSHEscape_PreservesInnerShellExpansion(t *testing.T) {
	t.Setenv("HOME", "/tmp/ssh-escape-home")
	command := `x="$(printf hi)"; printf '%s|%s|%s' "$x" "$HOME" 'lit'`
	out, err := exec.Command("sh", "-c", "sh -c "+sshEscape(command)).CombinedOutput()
	if err != nil {
		t.Fatalf("run escaped command: %v, output=%q", err, string(out))
	}
	got := strings.TrimSpace(string(out))
	if want := "hi|/tmp/ssh-escape-home|lit"; got != want {
		t.Fatalf("output=%q want %q", got, want)
	}
}

func TestSSHEscape_PreservesLiteralSingleQuotes(t *testing.T) {
	command := `printf '%s' 'a'\''b'`
	out, err := exec.Command("sh", "-c", "sh -c "+sshEscape(command)).CombinedOutput()
	if err != nil {
		t.Fatalf("run escaped command: %v, output=%q", err, string(out))
	}
	if got := string(out); got != "a'b" {
		t.Fatalf("output=%q want %q", got, "a'b")
	}
}

func TestSSHEscape_HelperProducesSingleQuotedWord(t *testing.T) {
	got := sshEscape(`echo "$HOME"`)
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Fatalf("escaped=%q", got)
	}
	if strings.Contains(got, `"$HOME"`) == false {
		t.Fatalf("escaped command lost inner content: %q", got)
	}
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("/bin/sh not available")
	}
}
