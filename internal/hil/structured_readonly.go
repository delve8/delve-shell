package hil

import (
	"path/filepath"
	"strings"

	"delve-shell/internal/config"
	"mvdan.cc/sh/v3/syntax"
)

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
	if filepath.Base(args[0]) != pol.Name {
		return false
	}
	rest := args[1:]

	i := 0
	g := pol.EffectiveGlobal()
	// Only consume leading globals when the policy declares global flags (any or allow-list).
	if g.Flags.IsAny() || g.Flags.IsAllowList() {
		for i < len(rest) {
			t := rest[i]
			if !strings.HasPrefix(t, "-") {
				break
			}
			if t == "--" {
				return false
			}
			n, ok := consumeFlag(rest, i, g.Flags)
			if !ok {
				return false
			}
			i += n
		}
	}

	return matchFromRoot(rest, i, pol.EffectiveRoot(), pol)
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
		merged := make([]config.AllowedOption, 0, len(ga)+len(local.AllowList()))
		merged = append(merged, ga...)
		merged = append(merged, local.AllowList()...)
		return config.NewFlagAllow(merged)
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

func matchFromRoot(rest []string, i int, root config.RootSpec, pol *config.ReadOnlyCLIPolicy) bool {
	if flatOpaqueTail(root) {
		return true
	}
	return matchNode(rest, i, root.Flags, root.Operands, root.Subcommands, true, pol)
}

func matchNode(rest []string, i int, flags config.FlagRule, operands config.OperandsRule, subs config.SubcommandMap, isKubectlStyleRoot bool, pol *config.ReadOnlyCLIPolicy) bool {
	eff := flagsWithGlobalAllowMerged(pol, flags)
	if len(subs) > 0 && i < len(rest) && !strings.HasPrefix(rest[i], "-") {
		if ch, ok := subs[rest[i]]; ok {
			return matchNode(rest, i+1, ch.EffectiveFlags(), ch.EffectiveOperands(), ch.Subcommands, false, pol)
		}
		if isKubectlStyleRoot || !operands.IsAny() {
			return false
		}
	}
	return consumeInterleaved(rest, i, eff, operands)
}

func consumeInterleaved(rest []string, i int, flags config.FlagRule, operands config.OperandsRule) bool {
	for i < len(rest) {
		t := rest[i]
		if strings.HasPrefix(t, "-") {
			if t == "--" {
				return false
			}
			n, ok := consumeFlag(rest, i, flags)
			if !ok {
				return false
			}
			i += n
			continue
		}
		if !operands.IsAny() {
			return false
		}
		i++
	}
	return true
}

func consumeFlag(args []string, i int, rule config.FlagRule) (n int, ok bool) {
	if rule.IsAny() {
		return consumeAnyFlag(args, i)
	}
	if rule.IsNone() {
		return 0, false
	}
	return consumeAllowListOpt(args, i, rule.AllowList())
}

func consumeAnyFlag(args []string, i int) (n int, ok bool) {
	t := args[i]
	if t == "--" {
		return 0, false
	}
	if strings.HasPrefix(t, "--") {
		if strings.Contains(t, "=") {
			return 1, true
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return 2, true
		}
		return 1, true
	}
	if strings.HasPrefix(t, "-") {
		if strings.Contains(t, "=") {
			return 1, true
		}
		if len(t) == 2 && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return 2, true
		}
		return 1, true
	}
	return 0, false
}

func consumeAllowListOpt(args []string, i int, opts []config.AllowedOption) (n int, ok bool) {
	t := args[i]
	if strings.HasPrefix(t, "--") {
		body := strings.TrimPrefix(t, "--")
		name, val, hasEq := strings.Cut(body, "=")
		for _, o := range opts {
			if o.Long == "" || name != o.Long {
				continue
			}
			if o.ValueRequired() {
				if hasEq {
					return 1, val != ""
				}
				if i+1 >= len(args) {
					return 0, false
				}
				return 2, true
			}
			return 1, true
		}
		return 0, false
	}
	if strings.HasPrefix(t, "-") && len(t) >= 2 && t[1] != '-' {
		for _, o := range opts {
			if o.Short == "" {
				continue
			}
			s := o.Short
			prefix := "-" + s
			if t == prefix {
				if o.ValueRequired() {
					if i+1 >= len(args) {
						return 0, false
					}
					return 2, true
				}
				return 1, true
			}
			eq := prefix + "="
			if strings.HasPrefix(t, eq) {
				if !o.ValueRequired() {
					return 0, false
				}
				return 1, len(t) > len(eq)
			}
		}
		return 0, false
	}
	return 0, false
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
	var base string
	if sa, ok := staticSimpleCommandArgs(seg); ok && len(sa) > 0 {
		base = argv0Base(sa[0])
	} else if pa, ok := permissiveSimpleArgv(seg); ok && len(pa) > 0 {
		base = argv0Base(pa[0])
	} else {
		return false
	}
	p, ok := w.cliByName[base]
	if !ok {
		return false
	}
	pp := p
	if p.PermissiveVarArgs() {
		if pa, ok := permissiveSimpleArgv(seg); ok && MatchReadOnlyCLIArgv(pa, &pp) {
			return true
		}
	}
	if sa, ok := staticSimpleCommandArgs(seg); ok && MatchReadOnlyCLIArgv(sa, &pp) {
		return true
	}
	return false
}

func segmentBareHelp(seg string) bool {
	s := strings.TrimSpace(seg)
	return s == "--help" || s == "-h"
}
