package hil

import (
	"regexp"
	"strings"
)

// SensitiveMatcher matches commands that may access sensitive paths (e.g. /etc/shadow, .env).
// Used to trigger user confirmation: "this command may access sensitive file(s)" with three choices (refuse / run+store / run+no store).
// Matching is path-only: any command string containing a sensitive path pattern matches (e.g. "cat /etc/shadow", "ls /etc/shadow", "cat /etc/shadow | wc").
type SensitiveMatcher struct {
	patterns []*regexp.Regexp
}

// NewSensitiveMatcher builds a matcher from the given path patterns (typically from config.LoadSensitivePatterns).
// Invalid patterns are skipped. Default list is in sensitive_patterns.yaml; see config.DefaultSensitivePatterns.
func NewSensitiveMatcher(patterns []string) *SensitiveMatcher {
	m := &SensitiveMatcher{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if re, err := regexp.Compile(p); err == nil {
			m.patterns = append(m.patterns, re)
		}
	}
	return m
}

// MayAccessSensitivePath returns true if the command string matches any sensitive path pattern (may access sensitive file(s)).
func (m *SensitiveMatcher) MayAccessSensitivePath(command string) bool {
	if m == nil {
		return false
	}
	command = strings.TrimSpace(command)
	for _, re := range m.patterns {
		if re != nil && re.MatchString(command) {
			return true
		}
	}
	return false
}
