package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestAllowStrict_ChainedCommand(t *testing.T) {
	// Build allowlist with only echo and tr (no openssl). Chain must require every segment to match.
	entries := []config.AllowlistEntry{
		{Pattern: `(^|\s)echo(\s|$)`},
		{Pattern: `(^|\s)tr(\s|$)`},
	}
	w := NewAllowlist(entries)

	// Single allowed command: allowed
	if !w.AllowStrict("echo hello") {
		t.Error("AllowStrict(echo hello) should be true")
	}
	// Chain where every segment is allowed: allowed
	if !w.AllowStrict("echo a && echo b") {
		t.Error("AllowStrict(echo a && echo b) should be true")
	}
	// Chain with one disallowed segment (openssl): not allowed
	if w.AllowStrict("openssl rand -base64 16 | tr -d '\\n' && echo") {
		t.Error("AllowStrict(openssl ... | tr ... && echo) should be false (openssl not on allowlist)")
	}
}

func TestContainsWriteRedirection(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ping -c 1 x.com", false},
		{"ping -c 1 x.com > /tmp/out", true},
		{"echo hello >> log.txt", true},
		{"cat a 2> err", true},
		{"cat a 2>> err", true},
		{"echo '> not redirect'", false},
		{"echo 'foo' > f", true},
		{"echo a >= b", false},
		{"echo a => b", false},
		{"true", false},
		{"", false},
		// discard / dup only: not treated as write redirection
		{"kubectl get pods 2>/dev/null", false},
		{"kubectl get pods 2> /dev/null", false},
		{"echo x >> /dev/null", false},
		{"cmd 2>&1", false},
		{"cmd >&2", false},
		{"cmd &> /dev/null", false},
		{"echo ok > /dev/null && echo x > /tmp/out", true},
		{"echo ok > /dev/nullx", true},
	}
	for _, tt := range tests {
		got := ContainsWriteRedirection(tt.cmd)
		if got != tt.want {
			t.Errorf("ContainsWriteRedirection(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestAllowStrict_SedWithoutInPlace(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	// Avoid unquoted | inside egrep pattern: splitShellChain does not honor quotes.
	cmd := "kubectl get pods 2>/dev/null | sed -n '1,260p' && echo '---' && kubectl get ns 2>/dev/null | egrep NAME || true"
	if ContainsWriteRedirection(cmd) {
		t.Fatal("ContainsWriteRedirection should be false for 2>/dev/null only")
	}
	if !w.AllowStrict(cmd) {
		t.Fatalf("AllowStrict(%q) want true", cmd)
	}
	if w.AllowStrict("sed -i.bak -n '1p' /etc/passwd") {
		t.Fatal("AllowStrict(sed -i...) should be false")
	}
	if w.AllowStrict("sed --in-place -n '1p' x") {
		t.Fatal("AllowStrict(sed --in-place...) should be false")
	}
}
