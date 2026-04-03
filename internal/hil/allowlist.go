package hil

import (
	"regexp"
	"strings"

	"delve-shell/internal/config"
)

var (
	sedLeadingCmd   = regexp.MustCompile(`^sed(\s|$)`)
	sedInPlaceFlags = regexp.MustCompile(`(?:^|\s)(?:-i(?:\.\S+)?(?:=[^\s]+)?|--in-place(?:=[^\s]+)?)(?:\s|$)`)

	jqLeadingCmd = regexp.MustCompile(`^jq(\s|$)`)
	// Reject loading a jq program from disk (-f / --from-file). Filter source is not analyzed.
	jqFromFileFlag = regexp.MustCompile(`(?:^|\s)-f(?:\s+|\s*=|$|\S)`)
	jqFromFileLong = regexp.MustCompile(`(?:^|\s)--from-file(?:\s+|\s*=|$)`)
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
// Outside single quotes, > not part of >= or => is a redirect unless the target is a discard/dup only:
// /dev/null, or &fd (e.g. 2>&1). Other targets still require user approval.
func ContainsWriteRedirection(command string) bool {
	const devNull = "/dev/null"
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
		if c != '>' {
			continue
		}
		if i+1 < len(command) && command[i+1] == '=' {
			continue // >=
		}
		if i > 0 && command[i-1] == '=' {
			continue // =>
		}
		opEnd := i + 1
		if opEnd < len(command) && command[opEnd] == '>' {
			opEnd++ // >>
		}
		targetStart := opEnd
		for targetStart < len(command) && (command[targetStart] == ' ' || command[targetStart] == '\t') {
			targetStart++
		}
		rest := command[targetStart:]
		if len(rest) >= len(devNull) && rest[:len(devNull)] == devNull {
			if len(rest) == len(devNull) || isRedirectTargetBoundary(rest[len(devNull)]) {
				i = targetStart + len(devNull) - 1
				continue
			}
		}
		if len(rest) > 0 && rest[0] == '&' {
			j := 1
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if j > 1 && (j == len(rest) || isRedirectTargetBoundary(rest[j])) {
				i = targetStart + j - 1
				continue
			}
		}
		return true
	}
	return false
}

func isRedirectTargetBoundary(b byte) bool {
	switch b {
	case ' ', '\t', '|', ';', '&', '#':
		return true
	default:
		return false
	}
}

// benignSedReadOnly is true for a segment whose command is sed without in-place flags (-i / --in-place).
// It does not try to rule out sed scripts that use the w command (write to file); that remains a residual risk.
func benignSedReadOnly(seg string) bool {
	s := strings.TrimSpace(seg)
	if !sedLeadingCmd.MatchString(s) {
		return false
	}
	return !sedInPlaceFlags.MatchString(s)
}

// benignJqReadOnly is true for jq without -f/--from-file (program from file). Stdin/stdout-only filters are assumed.
func benignJqReadOnly(seg string) bool {
	s := strings.TrimSpace(seg)
	if !jqLeadingCmd.MatchString(s) {
		return false
	}
	if jqFromFileFlag.MatchString(s) || jqFromFileLong.MatchString(s) {
		return false
	}
	return true
}

// Allow reports whether a single command string matches the allowlist (used per segment in AllowStrict).
func (w *Allowlist) Allow(command string) bool {
	for _, p := range w.patterns {
		if p.regex != nil && p.regex.MatchString(command) {
			return true
		}
	}
	return false
}

// splitIntoCommands splits a command into a flat list of single commands by pipeline (|) and chain (;, &&, ||).
func splitIntoCommands(command string) []string {
	parts := splitPipeline(command)
	var out []string
	for _, p := range parts {
		for _, sub := range splitShellChain(p) {
			if sub != "" {
				out = append(out, sub)
			}
		}
	}
	if len(out) == 0 {
		return []string{strings.TrimSpace(command)}
	}
	return out
}

// segmentAllowed is true when the segment matches the allowlist or is read-only sed (no -i / --in-place) or read-only jq (no -f/--from-file).
func (w *Allowlist) segmentAllowed(seg string) bool {
	return seg != "" && (w.Allow(seg) || benignSedReadOnly(seg) || benignJqReadOnly(seg))
}

// AllowStrict: for chained/pipeline commands, splits into segments and requires every segment to match the allowlist;
// for a single command, requires that one segment to match. All must match; no approval bypass by a single allowed token.
func (w *Allowlist) AllowStrict(command string) bool {
	segments := splitIntoCommands(command)
	for _, seg := range segments {
		if !w.segmentAllowed(seg) {
			return false
		}
	}
	return len(segments) > 0
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
			if !w.segmentAllowed(sub) {
				return false
			}
		}
	}
	return true
}
