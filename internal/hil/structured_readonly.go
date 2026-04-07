package hil

import (
	"path/filepath"
	"regexp"
	"strings"

	"delve-shell/internal/config"
	"mvdan.cc/sh/v3/syntax"
)

// exitStatusLiteral matches a decimal exit status for [benignExitReadOnly] (optional leading minus; bash wraps).
var exitStatusLiteral = regexp.MustCompile(`^-?[0-9]+$`)

// readOnlyCLIArg is one argv element for structured read-only matching. Opaque is true when the shell
// word was exactly one double-quoted simple parameter ("$var" or "${var}"); such a token may match only
// argv slots whose policy allows any value (flag values in allow-list mode, or operands: any).
type readOnlyCLIArg struct {
	lit    string
	opaque bool
}

func (a readOnlyCLIArg) literalOK() (string, bool) {
	if a.opaque {
		return "", false
	}
	return a.lit, true
}

func (a readOnlyCLIArg) isFlagToken() bool {
	if a.opaque {
		return false
	}
	return strings.HasPrefix(a.lit, "-")
}

func (a readOnlyCLIArg) subcommandKey() (string, bool) {
	return a.literalOK()
}

func wordToString(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	var b strings.Builder
	if err := syntax.NewPrinter().Print(&b, w); err != nil {
		return "", false
	}
	s := strings.TrimSpace(b.String())
	if s == "" {
		return "", false
	}
	return s, true
}

// wordToStaticString builds the runtime argv string for one parsed word: quote
// delimiters are stripped (single/double quotes). Words that still contain extglob
// or brace expansion after parsing are rejected here (see also [WordContainsExtGlob]).
func wordToStaticString(w *syntax.Word) (string, bool) {
	if w == nil || len(w.Parts) == 0 {
		return "", false
	}
	var b strings.Builder
	for _, p := range w.Parts {
		if !appendWordPartStatic(&b, p) {
			return "", false
		}
	}
	s := b.String()
	if strings.TrimSpace(s) == "" {
		return "", false
	}
	return s, true
}

func appendWordPartStatic(b *strings.Builder, p syntax.WordPart) bool {
	switch x := p.(type) {
	case *syntax.Lit:
		b.WriteString(x.Value)
		return true
	case *syntax.SglQuoted:
		b.WriteString(x.Value)
		return true
	case *syntax.DblQuoted:
		for _, sub := range x.Parts {
			if !appendWordPartStatic(b, sub) {
				return false
			}
		}
		return true
	case *syntax.BraceExp:
		return false
	default:
		return false
	}
}

// permissiveSimpleArgv parses a single simple command: argv[0] must be a literal word; later words may contain expansions.
func permissiveSimpleArgv(seg string) ([]string, bool) {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil, false
	}
	f, err := parseShell(seg)
	if err != nil || len(f.Stmts) != 1 {
		return nil, false
	}
	st := f.Stmts[0]
	if st == nil || st.Cmd == nil {
		return nil, false
	}
	ce, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(ce.Args) == 0 {
		return nil, false
	}
	args := make([]string, len(ce.Args))
	for i, w := range ce.Args {
		if w == nil {
			return nil, false
		}
		if i == 0 {
			args[i] = w.Lit()
			if args[i] == "" {
				return nil, false
			}
			continue
		}
		s, ok := wordToString(w)
		if !ok {
			return nil, false
		}
		args[i] = s
	}
	return args, true
}

// MatchReadOnlyCLIArgv checks argv (including argv0) against one read-only CLI policy.
// Args are normally produced by [staticSimpleCommandArgs]: argv0 is a single literal word;
// later argv entries are static (no parameter expansion, command substitution, etc.) but may
// combine mvdan Lit and quotes (e.g. kubectl -o 'custom-columns=...[?(@...)]...').
func MatchReadOnlyCLIArgv(args []string, pol *config.ReadOnlyCLIPolicy) bool {
	if len(args) == 0 || pol == nil {
		return false
	}
	ra := make([]readOnlyCLIArg, len(args))
	for i, s := range args {
		ra[i] = readOnlyCLIArg{lit: s}
	}
	return matchReadOnlyCLIArgs(ra, pol)
}

// matchReadOnlyCLIArgs is like [MatchReadOnlyCLIArgv] but supports opaque quoted simple parameters
// ("$ns") only in argv slots that accept any value (see [staticOrOpaqueSimpleCommandArgs]).
func matchReadOnlyCLIArgs(args []readOnlyCLIArg, pol *config.ReadOnlyCLIPolicy) bool {
	if len(args) == 0 || pol == nil {
		return false
	}
	name, ok := args[0].literalOK()
	if !ok || filepath.Base(name) != pol.Name {
		return false
	}
	rest := args[1:]

	i := 0
	g := pol.EffectiveGlobal()
	if g.Flags.IsAny() || g.Flags.IsAllowList() {
		for i < len(rest) {
			if !rest[i].isFlagToken() {
				break
			}
			t, _ := rest[i].literalOK()
			if t == "--" {
				return false
			}
			n, ok := consumeFlagArg(rest, i, g.Flags)
			if !ok {
				return false
			}
			i += n
		}
	}

	return matchFromRootArg(rest, i, pol.EffectiveRoot(), pol)
}

// flagsWithGlobalAllowMerged prepends the policy's global allow-list to a node's allow-list so the same
// option (e.g. -h/--help) need not be repeated under every subcommand: globals only consume leading
// tokens, while kubectl get --help places --help after the verb.
// For flags:none nodes, only the inherited globals apply (commands whose help lists no Options still
// accept persistent/global flags after the subcommand in kubectl).
func flagsWithGlobalAllowMerged(pol *config.ReadOnlyCLIPolicy, local config.FlagRule) config.FlagRule {
	if pol == nil {
		return local
	}
	g := pol.EffectiveGlobal()
	if !g.Flags.IsAllowList() {
		return local
	}
	ga := g.Flags.AllowList()
	if local.IsAllowList() {
		la := local.EffectiveConsumableAllowList()
		merged := make([]config.AllowedOption, 0, len(ga)+len(la))
		merged = append(merged, ga...)
		merged = append(merged, la...)
		out := config.NewFlagAllow(merged)
		if len(local.MustList()) > 0 {
			out = out.WithMust(local.MustList())
		}
		return out
	}
	if local.IsNone() {
		cp := append([]config.AllowedOption(nil), ga...)
		return config.NewFlagAllow(cp)
	}
	return local
}

// flatOpaqueTail is true when tail argv is not parsed into flags vs operands (any tokens allowed after globals).
func flatOpaqueTail(root config.RootSpec) bool {
	return len(root.Subcommands) == 0 && root.Flags.IsAny() && root.Operands.IsAny()
}

func matchFromRootArg(rest []readOnlyCLIArg, i int, root config.RootSpec, pol *config.ReadOnlyCLIPolicy) bool {
	if flatOpaqueTail(root) {
		return true
	}
	return matchNodeArg(rest, i, root.Flags, root.Operands, root.Subcommands, true, pol)
}

func matchNodeArg(rest []readOnlyCLIArg, i int, flags config.FlagRule, operands config.OperandsRule, subs config.SubcommandMap, isKubectlStyleRoot bool, pol *config.ReadOnlyCLIPolicy) bool {
	eff := flagsWithGlobalAllowMerged(pol, flags)
	must := eff.MustList()
	sat := make([]bool, len(must))
	i2, ok := consumeLeadingFlagsWithMust(rest, i, eff, sat)
	if !ok {
		return false
	}
	if len(subs) > 0 && i2 < len(rest) && !rest[i2].isFlagToken() {
		key, ok := rest[i2].subcommandKey()
		if !ok {
			return false
		}
		if ch, ok := subs[key]; ok {
			if !mustSliceAllTrue(sat) {
				return false
			}
			return matchNodeArg(rest, i2+1, ch.EffectiveFlags(), ch.EffectiveOperands(), ch.Subcommands, false, pol)
		}
		if isKubectlStyleRoot || !operands.IsAny() {
			return false
		}
	}
	return consumeInterleavedArg(rest, i2, eff, operands, sat)
}

// consumeLeadingFlagsWithMust consumes consecutive flag tokens at the current node and updates sat for
// flags.MustList(). It stops before the first non-flag token (including operands and subcommand names).
func consumeLeadingFlagsWithMust(rest []readOnlyCLIArg, i int, flags config.FlagRule, sat []bool) (int, bool) {
	must := flags.MustList()
	if len(sat) != len(must) {
		return i, false
	}
	for i < len(rest) && rest[i].isFlagToken() {
		t, _ := rest[i].literalOK()
		if t == "--" {
			return i, false
		}
		n, ok, allowIdx := consumeFlagArgIdx(rest, i, flags)
		if !ok {
			return i, false
		}
		if allowIdx >= 0 {
			effAllow := flags.EffectiveConsumableAllowList()
			if allowIdx < len(effAllow) {
				consumed := effAllow[allowIdx]
				for j := range must {
					if config.AllowedEntrySatisfiesMust(must[j], consumed) {
						sat[j] = true
					}
				}
			}
		}
		i += n
	}
	return i, true
}

func mustSliceAllTrue(sat []bool) bool {
	for _, b := range sat {
		if !b {
			return false
		}
	}
	return true
}

func consumeInterleavedArg(rest []readOnlyCLIArg, i int, flags config.FlagRule, operands config.OperandsRule, mustSat []bool) bool {
	must := flags.MustList()
	if len(mustSat) != len(must) {
		return false
	}
	sat := mustSat
	for i < len(rest) {
		if rest[i].isFlagToken() {
			t, _ := rest[i].literalOK()
			if t == "--" {
				return false
			}
			n, ok, allowIdx := consumeFlagArgIdx(rest, i, flags)
			if !ok {
				return false
			}
			if allowIdx >= 0 {
				effAllow := flags.EffectiveConsumableAllowList()
				if allowIdx < len(effAllow) {
					consumed := effAllow[allowIdx]
					for j := range must {
						if config.AllowedEntrySatisfiesMust(must[j], consumed) {
							sat[j] = true
						}
					}
				}
			}
			i += n
			continue
		}
		for j := range sat {
			if !sat[j] {
				return false
			}
		}
		if !operands.IsAny() {
			return false
		}
		i++
	}
	for j := range sat {
		if !sat[j] {
			return false
		}
	}
	return true
}

func consumeFlagArg(args []readOnlyCLIArg, i int, rule config.FlagRule) (n int, ok bool) {
	n, ok, _ = consumeFlagArgIdx(args, i, rule)
	return n, ok
}

func consumeFlagArgIdx(args []readOnlyCLIArg, i int, rule config.FlagRule) (n int, ok bool, allowIdx int) {
	allowIdx = -1
	if rule.IsAny() {
		n, ok = consumeAnyFlagArg(args, i)
		return n, ok, -1
	}
	if rule.IsNone() {
		return 0, false, -1
	}
	return consumeAllowListOptArgIdx(args, i, rule.EffectiveConsumableAllowList())
}

func consumeAnyFlagArg(args []readOnlyCLIArg, i int) (n int, ok bool) {
	t, ok := args[i].literalOK()
	if !ok {
		return 0, false
	}
	if t == "--" {
		return 0, false
	}
	if strings.HasPrefix(t, "--") {
		if strings.Contains(t, "=") {
			return 1, true
		}
		if i+1 < len(args) && !args[i+1].isFlagToken() {
			return 2, true
		}
		return 1, true
	}
	if strings.HasPrefix(t, "-") {
		if strings.Contains(t, "=") {
			return 1, true
		}
		if len(t) == 2 && i+1 < len(args) && !args[i+1].isFlagToken() {
			return 2, true
		}
		return 1, true
	}
	return 0, false
}

func consumeAllowListOptArgIdx(args []readOnlyCLIArg, i int, opts []config.AllowedOption) (n int, ok bool, allowIdx int) {
	allowIdx = -1
	t, litOK := args[i].literalOK()
	if !litOK {
		return 0, false, -1
	}
	if strings.HasPrefix(t, "--") {
		body := strings.TrimPrefix(t, "--")
		name, val, hasEq := strings.Cut(body, "=")
		for j, o := range opts {
			if o.Long == "" || name != o.Long {
				continue
			}
			if o.ValueRequired() {
				if hasEq {
					return 1, val != "", j
				}
				if i+1 >= len(args) {
					return 0, false, -1
				}
				return 2, true, j
			}
			return 1, true, j
		}
		return 0, false, -1
	}
	if strings.HasPrefix(t, "-") && len(t) >= 2 && t[1] != '-' {
		for j, o := range opts {
			if o.Short == "" {
				continue
			}
			s := o.Short
			prefix := "-" + s
			if t == prefix {
				if o.ValueRequired() {
					if i+1 >= len(args) {
						return 0, false, -1
					}
					return 2, true, j
				}
				return 1, true, j
			}
			eq := prefix + "="
			if strings.HasPrefix(t, eq) {
				if !o.ValueRequired() {
					return 0, false, -1
				}
				return 1, len(t) > len(eq), j
			}
		}
		return 0, false, -1
	}
	return 0, false, -1
}

// staticOrOpaqueSimpleCommandArgs parses one simple command into argv slots. After argv[0], a word that is
// exactly one double-quoted simple parameter ("$x" / "${x}") becomes an opaque placeholder; it matches
// policy only where the corresponding slot allows any value ([matchReadOnlyCLIArgs]).
func staticOrOpaqueSimpleCommandArgs(seg string) ([]readOnlyCLIArg, bool) {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil, false
	}
	f, err := parseShell(seg)
	if err != nil || len(f.Stmts) != 1 {
		return nil, false
	}
	st := f.Stmts[0]
	if st == nil || st.Cmd == nil {
		return nil, false
	}
	ce, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(ce.Args) == 0 {
		return nil, false
	}
	args := make([]readOnlyCLIArg, len(ce.Args))
	for i, w := range ce.Args {
		if w == nil {
			return nil, false
		}
		if i == 0 {
			lit := w.Lit()
			if lit == "" {
				return nil, false
			}
			args[i] = readOnlyCLIArg{lit: lit}
			continue
		}
		if WordContainsExtGlob(w) {
			return nil, false
		}
		if wordIsDoubleQuotedSimpleParamOnly(w) {
			args[i] = readOnlyCLIArg{opaque: true}
			continue
		}
		if wordContainsDisallowedShellExpansionForStructured(w) {
			return nil, false
		}
		s, ok := wordToStaticString(w)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, false
		}
		args[i] = readOnlyCLIArg{lit: s}
	}
	return args, true
}

// staticSimpleCommandArgs parses seg as one simple command: argv[0] must be a literal
// command name; later words use [wordToStaticString] (quote removal) and are accepted only
// when they contain no shell expansions ([WordContainsShellExpansion]) and no unquoted
// extended glob ([WordContainsExtGlob]), so argv used for policy matches real bash behavior
// for literal spans (e.g. JSONPath ?(...) must be inside single quotes).
func staticSimpleCommandArgs(seg string) ([]string, bool) {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return nil, false
	}
	f, err := parseShell(seg)
	if err != nil || len(f.Stmts) != 1 {
		return nil, false
	}
	st := f.Stmts[0]
	if st == nil || st.Cmd == nil {
		return nil, false
	}
	ce, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(ce.Args) == 0 {
		return nil, false
	}
	args := make([]string, len(ce.Args))
	for i, w := range ce.Args {
		if w == nil {
			return nil, false
		}
		if i == 0 {
			args[i] = w.Lit()
			if args[i] == "" {
				return nil, false
			}
			continue
		}
		if WordContainsShellExpansion(w) {
			return nil, false
		}
		if WordContainsExtGlob(w) {
			return nil, false
		}
		s, ok := wordToStaticString(w)
		if !ok || strings.TrimSpace(s) == "" {
			return nil, false
		}
		args[i] = s
	}
	return args, true
}

func (w *Allowlist) structuredLiteralSegmentOK(seg string) bool {
	if w == nil || len(w.cliByName) == 0 {
		return false
	}
	if qa, ok := staticOrOpaqueSimpleCommandArgs(seg); ok && len(qa) > 0 {
		name, lok := qa[0].literalOK()
		if !lok {
			return false
		}
		base := argv0Base(name)
		p, pok := w.cliByName[base]
		if !pok {
			return false
		}
		pp := p
		if p.PermissiveVarArgs() {
			pa, pok2 := permissiveSimpleArgv(seg)
			if pok2 && MatchReadOnlyCLIArgv(pa, &pp) {
				return true
			}
			return false
		}
		return matchReadOnlyCLIArgs(qa, &pp)
	}
	if pa, ok := permissiveSimpleArgv(seg); ok && len(pa) > 0 {
		base := argv0Base(pa[0])
		p, pok := w.cliByName[base]
		if pok && p.PermissiveVarArgs() {
			return MatchReadOnlyCLIArgv(pa, &p)
		}
	}
	return false
}

// benignExitReadOnly is true for bash "exit" or "exit N" where N is a literal decimal status, or "exit \"$var\""
// with a double-quoted simple parameter only. Rejects extra args, command substitution, and non-numeric literals.
func benignExitReadOnly(seg string) bool {
	seg = strings.TrimSpace(seg)
	if seg == "" {
		return false
	}
	f, err := parseShell(seg)
	if err != nil || len(f.Stmts) != 1 {
		return false
	}
	st := f.Stmts[0]
	if st == nil || st.Cmd == nil {
		return false
	}
	ce, ok := st.Cmd.(*syntax.CallExpr)
	if !ok || len(ce.Args) < 1 {
		return false
	}
	if ce.Args[0].Lit() == "" || argv0Base(ce.Args[0].Lit()) != "exit" {
		return false
	}
	if len(ce.Args) == 1 {
		return true
	}
	if len(ce.Args) != 2 {
		return false
	}
	w := ce.Args[1]
	if WordContainsExtGlob(w) {
		return false
	}
	if wordIsDoubleQuotedSimpleParamOnly(w) {
		return true
	}
	if wordContainsDisallowedShellExpansionForStructured(w) {
		return false
	}
	s, ok := wordToStaticString(w)
	if !ok || strings.TrimSpace(s) == "" {
		return false
	}
	return exitStatusLiteral.MatchString(strings.TrimSpace(s))
}

func segmentBareHelp(seg string) bool {
	s := strings.TrimSpace(seg)
	return s == "--help" || s == "-h"
}
