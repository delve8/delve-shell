package hil

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// shellUnwrapMax is how many times a bash/sh -c or -lc wrapper may be replaced by its script argument (one level only).
const shellUnwrapMax = 1

// CommandAllowsAutoApprove decides whether a shell command may run without explicit HIL approval.
//
// Policy:
//  1. Parse as Bash with mvdan.cc/sh/v3. If parsing fails, return false (require approval).
//  2. If the tree contains unsupported or high-risk constructs (e.g. coproc, Bats test declarations), return false.
//  3. Extract one text slice per execution unit (simple commands, tests, declare/export, including stmt-level redirects).
//     Command substitutions and process substitutions are walked so inner commands are included.
//     At most [shellUnwrapMax] times, a leading bash/sh invocation with -c or -lc is replaced by parsing its script argument only (nested bash -c inside that script is not unwrapped again).
//     A call whose command name is a function defined in the same parsed script is not checked as its own segment;
//     only the function body (and other statements) are checked—whether that function is invoked or not is ignored.
//  4. The full command string must pass [ContainsWriteRedirection] (stmt-level redirects included in the string).
//  5. Every extracted slice must pass the allowlist segment policy (same as AllowStrict per segment) and must not [ContainsWriteRedirection] on its own.
//
// An empty statement list after successful parse (e.g. only comments) yields no segments and returns true
// when steps 4–5 are vacuously satisfied.
func (w *Allowlist) CommandAllowsAutoApprove(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" || w == nil {
		return false
	}
	if ContainsWriteRedirection(command) {
		return false
	}
	f, err := parseShell(command)
	if err != nil {
		return false
	}
	segs, reject := collectAllowlistSegments(f, command, localFunctionNames(f), shellUnwrapMax)
	if reject {
		return false
	}
	seen := make(map[string]struct{}, len(segs))
	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		if _, ok := seen[seg]; ok {
			continue
		}
		seen[seg] = struct{}{}
		if ContainsWriteRedirection(seg) || !w.segmentAllowed(seg) {
			return false
		}
	}
	return true
}

func parseShell(command string) (*syntax.File, error) {
	p := syntax.NewParser(syntax.Variant(syntax.LangBash))
	r := strings.NewReader(command)
	return p.Parse(r, "")
}

func localFunctionNames(f *syntax.File) map[string]struct{} {
	m := make(map[string]struct{})
	syntax.Walk(f, func(n syntax.Node) bool {
		fd, ok := n.(*syntax.FuncDecl)
		if !ok || fd.Name == nil || fd.Name.Value == "" {
			return true
		}
		m[fd.Name.Value] = struct{}{}
		return true
	})
	return m
}

func collectAllowlistSegments(f *syntax.File, src string, localFuncs map[string]struct{}, unwrapLeft int) (segments []string, reject bool) {
	segs, rej := walkStmtList(f.Stmts, src, localFuncs, unwrapLeft)
	segments = append(segments, segs...)
	reject = reject || rej

	syntax.Walk(f, func(n syntax.Node) bool {
		switch x := n.(type) {
		case *syntax.CmdSubst:
			s, r := walkStmtList(x.Stmts, src, localFuncs, unwrapLeft)
			segments = append(segments, s...)
			reject = reject || r
		case *syntax.ProcSubst:
			s, r := walkStmtList(x.Stmts, src, localFuncs, unwrapLeft)
			segments = append(segments, s...)
			reject = reject || r
		}
		return true
	})
	return segments, reject
}

func walkStmtList(stmts []*syntax.Stmt, src string, localFuncs map[string]struct{}, unwrapLeft int) (segs []string, reject bool) {
	for _, s := range stmts {
		x, r := stmtSegments(s, src, localFuncs, unwrapLeft)
		segs = append(segs, x...)
		reject = reject || r
	}
	return segs, reject
}

func stmtSegments(st *syntax.Stmt, src string, localFuncs map[string]struct{}, unwrapLeft int) (segs []string, reject bool) {
	if st == nil || st.Cmd == nil {
		return nil, false
	}
	switch c := st.Cmd.(type) {
	case *syntax.BinaryCmd:
		sx, r1 := stmtSegments(c.X, src, localFuncs, unwrapLeft)
		sy, r2 := stmtSegments(c.Y, src, localFuncs, unwrapLeft)
		return append(sx, sy...), r1 || r2
	case *syntax.IfClause:
		return walkIfClause(c, src, localFuncs, unwrapLeft)
	case *syntax.Subshell:
		return walkStmtList(c.Stmts, src, localFuncs, unwrapLeft)
	case *syntax.Block:
		return walkStmtList(c.Stmts, src, localFuncs, unwrapLeft)
	case *syntax.WhileClause:
		s1, r1 := walkStmtList(c.Cond, src, localFuncs, unwrapLeft)
		s2, r2 := walkStmtList(c.Do, src, localFuncs, unwrapLeft)
		return append(s1, s2...), r1 || r2
	case *syntax.ForClause:
		return walkStmtList(c.Do, src, localFuncs, unwrapLeft)
	case *syntax.CaseClause:
		var out []string
		var rej bool
		for _, item := range c.Items {
			s, r := walkStmtList(item.Stmts, src, localFuncs, unwrapLeft)
			out = append(out, s...)
			rej = rej || r
		}
		return out, rej
	case *syntax.CallExpr:
		// Assignment-only simple command (no argv[0]); no separate external process name to allowlist.
		if len(c.Args) == 0 && len(c.Assigns) > 0 {
			return nil, false
		}
		if unwrapLeft > 0 {
			if script, ok := extractBashCScript(c); ok {
				innerF, err := parseShell(script)
				if err != nil {
					return nil, true
				}
				innerLocals := localFunctionNames(innerF)
				return walkStmtList(innerF.Stmts, script, innerLocals, unwrapLeft-1)
			}
		}
		if callName := simpleCallCommandName(c); callName != "" {
			if _, ok := localFuncs[callName]; ok {
				return nil, false
			}
		}
		return []string{byteSliceFromPos(src, st.Pos(), st.End())}, false
	case *syntax.TestClause:
		return []string{byteSliceFromPos(src, st.Pos(), st.End())}, false
	case *syntax.DeclClause:
		return []string{byteSliceFromPos(src, st.Pos(), st.End())}, false
	case *syntax.LetClause, *syntax.ArithmCmd:
		return nil, false
	case *syntax.FuncDecl:
		if c.Body == nil {
			return nil, false
		}
		return stmtSegments(c.Body, src, localFuncs, unwrapLeft)
	case *syntax.CoprocClause, *syntax.TestDecl:
		return nil, true
	case *syntax.TimeClause:
		if c.Stmt == nil {
			return nil, false
		}
		return stmtSegments(c.Stmt, src, localFuncs, unwrapLeft)
	default:
		return nil, true
	}
}

// extractBashCScript returns the script argument after a literal -c or -lc when the command is bash or sh.
func extractBashCScript(ce *syntax.CallExpr) (string, bool) {
	if ce == nil || len(ce.Args) < 3 {
		return "", false
	}
	if !isBashOrSh(ce.Args[0].Lit()) {
		return "", false
	}
	for i := 1; i < len(ce.Args)-1; i++ {
		lit := ce.Args[i].Lit()
		if lit != "-c" && lit != "-lc" {
			continue
		}
		script, ok := scriptFromWord(ce.Args[i+1])
		if !ok || script == "" {
			return "", false
		}
		return script, true
	}
	return "", false
}

// scriptFromWord returns the string passed to -c/-lc: unescaped for a single single-quoted word,
// otherwise re-serialized with the syntax printer (e.g. double quotes, unquoted).
func scriptFromWord(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	if len(w.Parts) == 1 {
		if sq, ok := w.Parts[0].(*syntax.SglQuoted); ok {
			return sq.Value, true
		}
	}
	var b strings.Builder
	if err := syntax.NewPrinter().Print(&b, w); err != nil || b.Len() == 0 {
		return "", false
	}
	return strings.TrimSpace(b.String()), true
}

func isBashOrSh(cmd0 string) bool {
	if cmd0 == "bash" || cmd0 == "sh" {
		return true
	}
	if i := strings.LastIndex(cmd0, "/"); i >= 0 {
		cmd0 = cmd0[i+1:]
	}
	return cmd0 == "bash" || cmd0 == "sh"
}

// simpleCallCommandName returns the first literal word of a simple command (the "name" position),
// or "" if the name is not a single literal (e.g. variable or expansion).
func simpleCallCommandName(ce *syntax.CallExpr) string {
	if ce == nil || len(ce.Args) == 0 {
		return ""
	}
	return ce.Args[0].Lit()
}

func walkIfClause(ic *syntax.IfClause, src string, localFuncs map[string]struct{}, unwrapLeft int) (segs []string, reject bool) {
	s1, r1 := walkStmtList(ic.Cond, src, localFuncs, unwrapLeft)
	s2, r2 := walkStmtList(ic.Then, src, localFuncs, unwrapLeft)
	segs = append(append(segs, s1...), s2...)
	reject = r1 || r2
	if ic.Else != nil {
		s3, r3 := walkIfClause(ic.Else, src, localFuncs, unwrapLeft)
		segs = append(segs, s3...)
		reject = reject || r3
	}
	return segs, reject
}

func byteSliceFromPos(src string, start, end syntax.Pos) string {
	i, j := int(start.Offset()), int(end.Offset())
	if i < 0 || j > len(src) || i > j {
		return ""
	}
	return strings.TrimSpace(src[i:j])
}
