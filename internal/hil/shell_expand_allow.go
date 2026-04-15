package hil

import (
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

func argv0Base(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	return filepath.Base(name)
}

// WordContainsShellExpansion reports whether w, under mvdan.cc/sh LangBash parsing, includes any of:
// parameter expansion ($var, ${...}, $1, …), command substitution, arithmetic expansion, or process substitution.
//
// Quoting (material to allowlist auto-approve):
//   - *syntax.SglQuoted spans are opaque: a dollar inside single quotes does not recurse and does not count.
//   - *syntax.DblQuoted spans recurse: "$HOME" yields ParamExp; whether "\$x" counts follows the parser’s
//     escape rules (often literal $, no ParamExp).
//
// Unknown word-part kinds are treated conservatively as expansion risk (same as before).
func WordContainsShellExpansion(w *syntax.Word) bool {
	return wordContainsShellExpansion(w)
}

// WordContainsExtGlob reports whether w includes bash extended glob syntax ?(...), *(...), etc.
// as parsed by mvdan (unquoted). Real bash applies pathname / extglob rules to such words, so
// the process argv may not match a naïvely reconstructed static string; structured auto-approve
// rejects these and requires explicit approval unless the user quotes the span so it becomes literal.
func WordContainsExtGlob(w *syntax.Word) bool {
	return wordContainsExtGlob(w)
}

func wordContainsExtGlob(w *syntax.Word) bool {
	if w == nil {
		return false
	}
	for _, p := range w.Parts {
		if wordPartContainsExtGlob(p) {
			return true
		}
	}
	return false
}

func wordPartContainsExtGlob(p syntax.WordPart) bool {
	switch x := p.(type) {
	case *syntax.ExtGlob:
		return true
	case *syntax.DblQuoted:
		for _, sub := range x.Parts {
			if wordPartContainsExtGlob(sub) {
				return true
			}
		}
		return false
	case *syntax.BraceExp:
		for _, ew := range x.Elems {
			if ew != nil && wordContainsExtGlob(ew) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func wordContainsShellExpansion(w *syntax.Word) bool {
	if w == nil || len(w.Parts) == 0 {
		return false
	}
	for _, p := range w.Parts {
		if wordPartContainsShellExpansion(p) {
			return true
		}
	}
	return false
}

func wordPartContainsShellExpansion(p syntax.WordPart) bool {
	switch x := p.(type) {
	case *syntax.Lit:
		return false
	case *syntax.SglQuoted:
		return false
	case *syntax.DblQuoted:
		for _, sub := range x.Parts {
			if wordPartContainsShellExpansion(sub) {
				return true
			}
		}
		return false
	case *syntax.ParamExp, *syntax.CmdSubst, *syntax.ArithmExp, *syntax.ProcSubst:
		return true
	case *syntax.ExtGlob:
		return false
	case *syntax.BraceExp:
		for _, ew := range x.Elems {
			if wordContainsShellExpansion(ew) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// commandAllowsShellExpansionInArgsPastArgv0 is for argv[0] values where trailing args may legally
// contain $ that parses as ParamExp but is not “kubectl-style dynamic argv” risk (e.g. awk field refs in
// double quotes). Those commands still need their own segment policy (e.g. benignAwkReadOnly).
func commandAllowsShellExpansionInArgsPastArgv0(cmd0 string) bool {
	return awkFamilyInvokerBase(argv0Base(cmd0))
}

func callExprArgsContainShellExpansion(args []*syntax.Word) bool {
	for _, w := range args {
		if wordContainsShellExpansion(w) {
			return true
		}
	}
	return false
}

// isSimpleParamExp is true for $name or ${name} with no slice, default, pattern ops, etc.
func isSimpleParamExp(pe *syntax.ParamExp) bool {
	if pe == nil || pe.Param == nil || pe.Param.Value == "" {
		return false
	}
	if pe.Excl || pe.Length || pe.Width || pe.Index != nil || pe.Slice != nil || pe.Repl != nil || pe.Exp != nil {
		return false
	}
	if pe.Names != 0 {
		return false
	}
	// ${a[i]} / $a[1] style expansions are not a plain name.
	if pe.Index != nil {
		return false
	}
	return true
}

// wordIsDoubleQuotedSimpleParamOnly is true when w is exactly one double-quoted simple parameter expansion.
func wordIsDoubleQuotedSimpleParamOnly(w *syntax.Word) bool {
	if w == nil || len(w.Parts) != 1 {
		return false
	}
	dq, ok := w.Parts[0].(*syntax.DblQuoted)
	if !ok || len(dq.Parts) != 1 {
		return false
	}
	pe, ok := dq.Parts[0].(*syntax.ParamExp)
	if !ok {
		return false
	}
	return isSimpleParamExp(pe)
}

// wordIsDoubleQuotedCmdSubstOnly is true when w is exactly one double-quoted command substitution:
// "$(...)". The command body is checked separately by the shell allowlist walker.
func wordIsDoubleQuotedCmdSubstOnly(w *syntax.Word) bool {
	if w == nil || len(w.Parts) != 1 {
		return false
	}
	dq, ok := w.Parts[0].(*syntax.DblQuoted)
	if !ok || len(dq.Parts) != 1 {
		return false
	}
	_, ok = dq.Parts[0].(*syntax.CmdSubst)
	return ok
}

// wordIsFlagWithDoubleQuotedCmdSubstValue is true for a single argv word that statically names a flag
// and attaches a quoted command substitution as its value, e.g. --name="$(...)" or -n="$(...)".
func wordIsFlagWithDoubleQuotedCmdSubstValue(w *syntax.Word) (flagToken string, ok bool) {
	if w == nil || len(w.Parts) != 2 {
		return "", false
	}
	lit, ok := w.Parts[0].(*syntax.Lit)
	if !ok || !validFlagAssignmentPrefix(lit.Value) {
		return "", false
	}
	dq, ok := w.Parts[1].(*syntax.DblQuoted)
	if !ok || len(dq.Parts) != 1 {
		return "", false
	}
	if _, ok := dq.Parts[0].(*syntax.CmdSubst); !ok {
		return "", false
	}
	return lit.Value, true
}

func validFlagAssignmentPrefix(s string) bool {
	if !strings.HasSuffix(s, "=") {
		return false
	}
	name := strings.TrimSuffix(s, "=")
	if name == "" || name == "-" || name == "--" {
		return false
	}
	return strings.HasPrefix(name, "-")
}

// wordContainsDisallowedShellExpansionForStructured is true when w has shell expansions that are not
// a lone double-quoted simple parameter (e.g. "$ns") or a quoted command substitution in a form that
// can be treated as an opaque value. Those placeholders may still match policy only if the
// corresponding argv slot allows the dynamic value (see [matchReadOnlyCLIArgs]).
func wordContainsDisallowedShellExpansionForStructured(w *syntax.Word) bool {
	if !wordContainsShellExpansion(w) {
		return false
	}
	if wordIsDoubleQuotedSimpleParamOnly(w) {
		return false
	}
	if wordIsDoubleQuotedCmdSubstOnly(w) {
		return false
	}
	if _, ok := wordIsFlagWithDoubleQuotedCmdSubstValue(w); ok {
		return false
	}
	return true
}

// wordContainsDisallowedShellExpansionForReadBuiltin is stricter than
// [wordContainsDisallowedShellExpansionForStructured]: read has no external argv policy to prove a
// command substitution is a harmless value, so quoted "$(...)" stays disallowed there.
func wordContainsDisallowedShellExpansionForReadBuiltin(w *syntax.Word) bool {
	if !wordContainsShellExpansion(w) {
		return false
	}
	if wordIsDoubleQuotedSimpleParamOnly(w) {
		return false
	}
	return true
}

func callExprArgsContainDisallowedExpansionForStructured(args []*syntax.Word) bool {
	for _, w := range args {
		if wordContainsDisallowedShellExpansionForStructured(w) {
			return true
		}
	}
	return false
}

func callExprArgsContainDisallowedExpansionForReadBuiltin(args []*syntax.Word) bool {
	for _, w := range args {
		if wordContainsDisallowedShellExpansionForReadBuiltin(w) {
			return true
		}
	}
	return false
}
