package hil

import (
	"regexp"
	"strings"

	"delve-shell/internal/config"
)

// Allowlist is a config-based allowlist matcher.
type Allowlist struct {
	patterns []compiledEntry
}

type compiledEntry struct {
	regex *regexp.Regexp
}

// NewAllowlist builds a matcher from allowlist entries; each Pattern is a regex, invalid ones are ignored.
func NewAllowlist(entries []config.AllowlistEntry) *Allowlist {
	w := &Allowlist{}
	for _, e := range entries {
		if re, err := regexp.Compile(e.Pattern); err == nil {
			w.patterns = append(w.patterns, compiledEntry{regex: re})
		}
	}
	return w
}

// ContainsWriteRedirection reports whether the command contains write redirection (>, >>, 2>, etc.).
// Heuristic: > outside single quotes and not >= or => is treated as write redirection and requires user approval.
func ContainsWriteRedirection(command string) bool {
	inSingle := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		if c == '\'' {
			inSingle = !inSingle
			continue
		}
		if inSingle {
			continue
		}
		if c == '>' {
			nextEq := i+1 < len(command) && command[i+1] == '='
			prevEq := i > 0 && command[i-1] == '='
			if !nextEq && !prevEq {
				return true
			}
		}
	}
	return false
}

// Allow reports whether the full command (or script) matches the allowlist; if so, no user approval needed.
func (w *Allowlist) Allow(command string) bool {
	for _, p := range w.patterns {
		if p.regex != nil && p.regex.MatchString(command) {
			return true
		}
	}
	return false
}

// splitPipeline splits the command by pipe |, ignoring | inside quotes.
func splitPipeline(command string) []string {
	var parts []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			b.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			b.WriteByte(c)
		case c == '|' && !inSingle && !inDouble:
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, strings.TrimSpace(b.String()))
	}
	return parts
}

// splitShellChain splits a segment by ; && || into subcommands (strict check so "cat x; rm -rf /" is not allowed by cat).
func splitShellChain(segment string) []string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return nil
	}
	// simple split by ; and && || (no quote handling), trim each segment
	var out []string
	for _, s := range strings.FieldsFunc(segment, func(r rune) bool {
		return r == ';' || r == '&' || r == '|'
	}) {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{segment}
	}
	return out
}

// AllowPipeline if the command has pipes, splits into subcommands; returns true only when every subcommand matches allowlist.
// Each segment is further split by ; && || so "cat x; rm -rf /" is not allowed as a whole.
func (w *Allowlist) AllowPipeline(command string) bool {
	parts := splitPipeline(command)
	if len(parts) <= 1 {
		return false
	}
	for _, part := range parts {
		for _, sub := range splitShellChain(part) {
			if sub == "" || !w.Allow(sub) {
				return false
			}
		}
	}
	return true
}
