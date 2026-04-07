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
	switch argv0Base(cmd0) {
	case "awk":
		return true
	default:
		return false
	}
}

func callExprArgsContainShellExpansion(args []*syntax.Word) bool {
	for _, w := range args {
		if wordContainsShellExpansion(w) {
			return true
		}
	}
	return false
}
