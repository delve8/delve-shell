package hil

import (
	"sort"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// expansionPolicyRiskSpans lists byte ranges that explain [collectAllowlistSegments] rejection when a
// non-permissive argv0 disallows shell expansions in trailing arguments: each ParamExp / CmdSubst /
// ArithmExp / ProcSubst in those args, plus literal names after `read` (skipping leading -options) and
// the iterator name in `for NAME in ...`.
//
// It returns nil unless there is at least one expansion site under such a call (so dynamic argv[0] and
// other reject reasons still fall back to full-line Risk in [Allowlist.CommandAutoApproveHighlight]).
func expansionPolicyRiskSpans(command string, f *syntax.File, localFuncs map[string]struct{}, argv0AllowsVarArgs func(string) bool) []allowSeg {
	n := len(command)
	if f == nil || n == 0 {
		return nil
	}
	var argSpans, defSpans []allowSeg
	syntax.Walk(f, func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.ForClause:
			if wi, ok := x.Loop.(*syntax.WordIter); ok && wi.Name != nil {
				if sg, ok := litByteSpan(n, wi.Name); ok {
					defSpans = append(defSpans, sg)
				}
			}
		case *syntax.CallExpr:
			appendReadVarDefSpans(n, x, &defSpans)
			callName := simpleCallCommandName(x)
			if len(x.Args) == 0 || callName == "" {
				return true
			}
			if _, ok := localFuncs[callName]; ok {
				return true
			}
			if testOrBracketBuiltin(callName) || shellBuiltinNoAllowlistSegment(callName) {
				return true
			}
			allowVar := argv0AllowsVarArgs != nil && argv0AllowsVarArgs(callName)
			if commandAllowsShellExpansionInArgsPastArgv0(callName) {
				allowVar = true
			}
			if allowVar {
				return true
			}
			for _, w := range x.Args[1:] {
				if !wordContainsDisallowedShellExpansionForStructured(w) {
					continue
				}
				appendExpansionRiskWordParts(n, w, &argSpans)
			}
		}
		return true
	})
	if len(argSpans) == 0 {
		return nil
	}
	all := make([]allowSeg, 0, len(argSpans)+len(defSpans))
	all = append(all, argSpans...)
	all = append(all, defSpans...)
	return mergeAllowSegs(all, n)
}

func appendReadVarDefSpans(n int, ce *syntax.CallExpr, spans *[]allowSeg) {
	if ce == nil || len(ce.Args) < 2 {
		return
	}
	if argv0Base(simpleCallCommandName(ce)) != "read" {
		return
	}
	for i := 1; i < len(ce.Args); i++ {
		w := ce.Args[i]
		if w == nil || len(w.Parts) != 1 {
			continue
		}
		lit, ok := w.Parts[0].(*syntax.Lit)
		if !ok {
			continue
		}
		v := lit.Value
		if v == "" || strings.HasPrefix(v, "-") {
			continue
		}
		if sg, ok := litByteSpan(n, lit); ok {
			*spans = append(*spans, sg)
		}
	}
}

func litByteSpan(n int, lit *syntax.Lit) (allowSeg, bool) {
	if lit == nil {
		return allowSeg{}, false
	}
	return nodeByteSpan(lit, n)
}

func nodeByteSpan(n syntax.Node, maxLen int) (allowSeg, bool) {
	if n == nil {
		return allowSeg{}, false
	}
	i := int(n.Pos().Offset())
	j := int(n.End().Offset())
	if i < 0 || j > maxLen || i > j {
		return allowSeg{}, false
	}
	return allowSeg{start: i, end: j}, true
}

func appendExpansionRiskWordParts(n int, w *syntax.Word, spans *[]allowSeg) {
	if w == nil {
		return
	}
	for _, p := range w.Parts {
		appendExpansionRiskWordPart(n, p, spans)
	}
}

func appendExpansionRiskWordPart(n int, p syntax.WordPart, spans *[]allowSeg) {
	switch x := p.(type) {
	case *syntax.Lit, *syntax.SglQuoted:
		return
	case *syntax.ParamExp, *syntax.CmdSubst, *syntax.ArithmExp, *syntax.ProcSubst:
		if sg, ok := nodeByteSpan(x, n); ok {
			*spans = append(*spans, sg)
		}
	case *syntax.DblQuoted:
		for _, sub := range x.Parts {
			appendExpansionRiskWordPart(n, sub, spans)
		}
	case *syntax.BraceExp:
		for _, ew := range x.Elems {
			if ew != nil {
				appendExpansionRiskWordParts(n, ew, spans)
			}
		}
	case *syntax.ExtGlob:
		if sg, ok := nodeByteSpan(x, n); ok {
			*spans = append(*spans, sg)
		}
	}
}

func mergeAllowSegs(spans []allowSeg, n int) []allowSeg {
	if len(spans) == 0 {
		return nil
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end < spans[j].end
	})
	out := []allowSeg{spans[0]}
	for _, s := range spans[1:] {
		last := &out[len(out)-1]
		if s.start <= last.end {
			if s.end > last.end {
				last.end = s.end
			}
		} else {
			out = append(out, s)
		}
	}
	for i := range out {
		if out[i].start < 0 {
			out[i].start = 0
		}
		if out[i].end > n {
			out[i].end = n
		}
		if out[i].start > out[i].end {
			out[i].end = out[i].start
		}
	}
	return out
}
