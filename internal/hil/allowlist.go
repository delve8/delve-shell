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

	sortLeadingCmd = regexp.MustCompile(`^sort(\s|$)`)
	// Reject -o / --output (write sorted output to a file; GNU coreutils).
	sortOutputShort = regexp.MustCompile(`(?:^|\s)-o(?:\s|$|\S)`)
	sortOutputLong  = regexp.MustCompile(`(?:^|\s)--output(?:=\S*|\s+|$)`)
)

// Allowlist is a config-based allowlist matcher (allowlist.yaml schema v2: commands map by argv0 basename).
type Allowlist struct {
	cliByName       map[string]config.ReadOnlyCLIPolicy
	permissiveArgv0 map[string]struct{}
}

// NewAllowlist builds a matcher from a loaded allowlist. If ld is nil, uses [config.DefaultLoadedAllowlist].
func NewAllowlist(ld *config.LoadedAllowlist) *Allowlist {
	if ld == nil {
		ld = config.DefaultLoadedAllowlist()
	}
	pol := config.NormalizeReadOnlyCLIPolicies(ld.Commands)
	per := make(map[string]struct{})
	for name, p := range pol {
		if p.PermissiveVarArgs() {
			per[name] = struct{}{}
		}
	}
	return &Allowlist{cliByName: pol, permissiveArgv0: per}
}

func (w *Allowlist) argv0PermitsVarArgs(argv0 string) bool {
	if w == nil {
		return false
	}
	_, ok := w.permissiveArgv0[argv0Base(argv0)]
	return ok
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

// benignSortReadOnly is true for sort without -o/--output (those write to a path).
func benignSortReadOnly(seg string) bool {
	s := strings.TrimSpace(seg)
	if !sortLeadingCmd.MatchString(s) {
		return false
	}
	if sortOutputShort.MatchString(s) || sortOutputLong.MatchString(s) {
		return false
	}
	return true
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

// segmentAllowed is true when the segment matches structured read-only CLI policy, bare --help/-h,
// or is read-only sed (no -i / --in-place), read-only jq (no -f/--from-file), read-only awk
// (GoAWK parse: no print redirect, no system(), no cmd|getline), read-only sort (no -o/--output),
// or benign "exit" / "exit N" ([benignExitReadOnly]).
func (w *Allowlist) segmentAllowed(seg string) bool {
	return seg != "" && (segmentBareHelp(seg) || w.structuredLiteralSegmentOK(seg) || benignSedReadOnly(seg) || benignJqReadOnly(seg) || benignAwkReadOnly(seg) || benignSortReadOnly(seg) || benignExitReadOnly(seg))
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
