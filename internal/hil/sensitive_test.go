package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestSensitiveMatcher_MayAccessSensitivePath(t *testing.T) {
	m := NewSensitiveMatcher(config.DefaultSensitivePatterns())

	tests := []struct {
		cmd    string
		expect bool
	}{
		{"cat /etc/shadow", true},
		{"ls /etc/shadow", true},
		{"cat /etc/shadow | wc", true},
		{"head -1 /etc/shadow", true},
		{"cat /etc/gpg/foo", true},
		{"cat .env", true},
		{"cat /tmp/app/.env", true},
		{"cat /path/to/id_rsa", true},
		{"  cat /etc/shadow  ", true},
		{"ls /etc", false},
		{"cat /etc/passwd", false},
		{"echo hello", false},
	}
	for _, tt := range tests {
		got := m.MayAccessSensitivePath(tt.cmd)
		if got != tt.expect {
			t.Errorf("MayAccessSensitivePath(%q) = %v, want %v", tt.cmd, got, tt.expect)
		}
	}
}

func TestSensitiveMatcher_Nil(t *testing.T) {
	var m *SensitiveMatcher
	if m.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("nil matcher should not match")
	}
}

func TestSensitiveMatcher_EmptyPatterns(t *testing.T) {
	m := NewSensitiveMatcher(nil)
	if m.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("nil patterns should not match")
	}
	m2 := NewSensitiveMatcher([]string{})
	if m2.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("empty patterns should not match")
	}
}

func TestSensitiveMatcher_InvalidPatternSkipped(t *testing.T) {
	m := NewSensitiveMatcher([]string{`[invalid`, `/etc/shadow\b`})
	if !m.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("valid pattern should match when invalid is skipped")
	}
	if m.MayAccessSensitivePath("cat /etc/passwd") {
		t.Error("should not match")
	}
}

func TestSensitiveMatcher_WhitespacePatternSkipped(t *testing.T) {
	m := NewSensitiveMatcher([]string{"  ", "\t", "", "/etc/shadow\\b"})
	if !m.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("non-empty pattern should match")
	}
}

func TestSensitiveMatcher_CustomPattern(t *testing.T) {
	patterns := append([]string{}, config.DefaultSensitivePatterns()...)
	patterns = append(patterns, `/my/secret/path`)
	m := NewSensitiveMatcher(patterns)
	if !m.MayAccessSensitivePath("cat /my/secret/path/file") {
		t.Error("custom pattern should match")
	}
	if !m.MayAccessSensitivePath("cat /etc/shadow") {
		t.Error("default /etc/shadow should still match when custom patterns are present")
	}
}
