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
	}
	for _, tt := range tests {
		got := ContainsWriteRedirection(tt.cmd)
		if got != tt.want {
			t.Errorf("ContainsWriteRedirection(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}
