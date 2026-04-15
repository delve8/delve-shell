package execenv

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func TestParseSSHConnectTarget_AllowsImplicitUserForProxyJump(t *testing.T) {
	t.Setenv("USER", "alice")
	got, err := parseSSHConnectTarget("jump.example.com:2222", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.user != "alice" {
		t.Fatalf("user=%q want alice", got.user)
	}
	if got.hostPort != "jump.example.com:2222" {
		t.Fatalf("hostPort=%q want jump.example.com:2222", got.hostPort)
	}
}

func TestResolveProxyJumpTarget_UsesSSHConfigAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeSSHConfigForExecenvTest(t, home, `
Host bastion
  HostName bastion.example.com
  User jump
  Port 2201
  IdentityFile ~/.ssh/bastion_key
`)

	target, identityFile, err := resolveProxyJumpTarget("bastion")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target != "jump@bastion.example.com:2201" {
		t.Fatalf("target=%q want jump@bastion.example.com:2201", target)
	}
	if identityFile != filepath.Join(home, ".ssh", "bastion_key") {
		t.Fatalf("identityFile=%q", identityFile)
	}
}

func TestResolveProxyJumpTarget_RejectsMultiHop(t *testing.T) {
	if _, _, err := resolveProxyJumpTarget("jump1,jump2"); err == nil {
		t.Fatal("expected multi-hop ProxyJump to fail")
	}
}

func TestResolveProxyJumpTarget_RejectsNestedProxyJumpAlias(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeSSHConfigForExecenvTest(t, home, `
Host bastion
  HostName bastion.example.com
  User jump
  ProxyJump outer

Host outer
  HostName outer.example.com
  User jump
`)

	if _, _, err := resolveProxyJumpTarget("bastion"); err == nil {
		t.Fatal("expected nested ProxyJump alias to fail")
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

func TestSSHExecutorSetTransportIssueHandler_ReplaysCurrentIssue(t *testing.T) {
	exec := &SSHExecutor{transportIssue: "disconnected"}
	got := ""
	exec.SetTransportIssueHandler(func(issue string) {
		got = issue
	})
	if got != exec.transportIssue {
		t.Fatalf("issue=%q want %q", got, exec.transportIssue)
	}
}

func TestSSHExecutorReportTransportIssue_Dedupes(t *testing.T) {
	exec := &SSHExecutor{}
	var got []string
	exec.SetTransportIssueHandler(func(issue string) {
		got = append(got, issue)
	})
	exec.reportTransportIssue("lost")
	exec.reportTransportIssue("lost")
	exec.reportTransportIssue("")
	if want := []string{"lost", ""}; len(got) != len(want) {
		t.Fatalf("calls=%v want %v", got, want)
	}
	if got[0] != "lost" || got[1] != "" {
		t.Fatalf("calls=%v want [lost \"\"]", got)
	}
}

func TestSSHConnectionIssueSummary(t *testing.T) {
	if got := SSHConnectionIssueSummary(errors.New("x")); got != "" {
		t.Fatalf("summary=%q want empty", got)
	}
	if got := SSHConnectionIssueSummary(&SSHConnectionError{Op: "run", Err: errors.New("boom")}); got != "disconnected" {
		t.Fatalf("summary=%q want disconnected", got)
	}
}

func writeSSHConfigForExecenvTest(t *testing.T, home string, content string) {
	t.Helper()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
