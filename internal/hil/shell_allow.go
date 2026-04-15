package hil

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// shellUnwrapMax is how many times a bash/sh -c or -lc wrapper may be replaced by its script argument (one level only).
const shellUnwrapMax = 1

// allowSeg is a half-open byte range [start,end) into the root trimmed command string.
type allowSeg struct {
	start, end int
}

type walkCtx struct {
	root string
	src  string
	base int
}

func (c walkCtx) segFromStmt(st *syntax.Stmt) (allowSeg, bool) {
	if st == nil {
		return allowSeg{}, false
	}
	i := int(st.Pos().Offset())
	j := int(st.End().Offset())
	if i < 0 || j > len(c.src) || i > j {
		return allowSeg{}, false
	}
	ai, aj := c.base+i, c.base+j
	if ai < 0 {
		ai = 0
	}
	if aj > len(c.root) {
		aj = len(c.root)
	}
	if ai > aj {
		return allowSeg{}, false
	}
	return allowSeg{start: ai, end: aj}, true
}

// CommandAllowsAutoApprove decides whether a shell command may run without explicit HIL approval.
//
// Policy:
//  1. Parse as Bash with mvdan.cc/sh/v3. If parsing fails, return false (require approval).
//  2. If the tree contains unsupported or high-risk constructs (e.g. coproc, Bats test declarations), return false.
//  3. Extract one text slice per execution unit (simple commands, declare/export, including stmt-level redirects).
//     TestClause ([ and [[ ... ]]), CallExpr [ and test (POSIX spelling), and ":" do not produce their own allowlist segment;
//     command/process substitutions under those nodes and under the full parse tree are still walked so inner commands are included.
//     At most [shellUnwrapMax] times, a leading bash/sh invocation with -c or -lc is replaced by parsing its script argument only (nested bash -c inside that script is not unwrapped again).
//     A call whose command name is a function defined in the same parsed script is not checked as its own segment;
//     only the function body (and other statements) are checked—whether that function is invoked or not is ignored.
//  4. The full command string must pass [ContainsWriteRedirection] (stmt-level redirects included in the string).
//  5. Every extracted slice must pass the allowlist segment policy (same as AllowStrict per segment) and must not [ContainsWriteRedirection] on its own.
//  6. For a simple command, argv[0] must be a single literal word (no dynamic command name). If the loaded allowlist
//     does not mark that basename as permissive (flat flags:any + operands:any policy), any argument after argv[0] must
//     contain no shell expansions ($, ${}, $(), arithmetic, etc.) and no unquoted bash extended glob (?(...), *(...), …);
//     permissive read-only utilities may use variables in arguments.
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
	varArg := func(name string) bool { return w.argv0PermitsVarArgs(name) }
	segs, _, reject := collectAllowlistSegments(f, command, localFunctionNames(f), shellUnwrapMax, varArg)
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

func collectAllowlistSegments(f *syntax.File, root string, localFuncs map[string]struct{}, unwrapLeft int, argv0AllowsVarArgs func(string) bool) (segments []string, ranges []allowSeg, reject bool) {
	ctx := walkCtx{root: root, src: root, base: 0}
	segs, rej := walkStmtList(f.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	ranges = append(ranges, segs...)
	reject = rej

	s, r := substSegsFromNode(f, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	ranges = append(ranges, s...)
	reject = reject || r

	for _, rg := range ranges {
		if rg.start < 0 || rg.end > len(root) || rg.start > rg.end {
			continue
		}
		segments = append(segments, strings.TrimSpace(root[rg.start:rg.end]))
	}
	return segments, ranges, reject
}

// substSegsFromNode collects command/process substitution bodies under n (for nodes that do not emit their own allowlist segment).
func substSegsFromNode(n syntax.Node, ctx walkCtx, localFuncs map[string]struct{}, unwrapLeft int, argv0AllowsVarArgs func(string) bool) (segments []allowSeg, reject bool) {
	syntax.Walk(n, func(node syntax.Node) bool {
		switch x := node.(type) {
		case *syntax.CmdSubst:
			s, r := walkStmtList(x.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
			segments = append(segments, s...)
			reject = reject || r
		case *syntax.ProcSubst:
			s, r := walkStmtList(x.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
			segments = append(segments, s...)
			reject = reject || r
		}
		return true
	})
	return segments, reject
}

func walkStmtList(stmts []*syntax.Stmt, ctx walkCtx, localFuncs map[string]struct{}, unwrapLeft int, argv0AllowsVarArgs func(string) bool) (segs []allowSeg, reject bool) {
	for _, s := range stmts {
		x, r := stmtSegments(s, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		segs = append(segs, x...)
		reject = reject || r
	}
	return segs, reject
}

func stmtSegments(st *syntax.Stmt, ctx walkCtx, localFuncs map[string]struct{}, unwrapLeft int, argv0AllowsVarArgs func(string) bool) (segs []allowSeg, reject bool) {
	if st == nil || st.Cmd == nil {
		return nil, false
	}
	switch c := st.Cmd.(type) {
	case *syntax.BinaryCmd:
		sx, r1 := stmtSegments(c.X, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		sy, r2 := stmtSegments(c.Y, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		return append(sx, sy...), r1 || r2
	case *syntax.IfClause:
		return walkIfClause(c, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.Subshell:
		return walkStmtList(c.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.Block:
		return walkStmtList(c.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.WhileClause:
		s1, r1 := walkStmtList(c.Cond, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		s2, r2 := walkStmtList(c.Do, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		return append(s1, s2...), r1 || r2
	case *syntax.ForClause:
		return walkStmtList(c.Do, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.CaseClause:
		var out []allowSeg
		var rej bool
		for _, item := range c.Items {
			s, r := walkStmtList(item.Stmts, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
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
			script, word, ok := extractBashCScriptAndWord(c)
			if ok {
				innerF, err := parseShell(script)
				if err != nil {
					return nil, true
				}
				base, ok := scriptContentBaseInRoot(ctx.root, word, script)
				if !ok {
					return nil, true
				}
				innerLocals := localFunctionNames(innerF)
				innerCtx := walkCtx{root: ctx.root, src: script, base: base}
				return walkStmtList(innerF.Stmts, innerCtx, innerLocals, unwrapLeft-1, argv0AllowsVarArgs)
			}
		}
		callName := simpleCallCommandName(c)
		if len(c.Args) > 0 && callName == "" {
			// Dynamic argv[0]: cannot bind to allowlist policy.
			return nil, true
		}
		if len(c.Args) > 0 && callName != "" {
			if _, ok := localFuncs[callName]; ok {
				return nil, false
			}
			if testOrBracketBuiltin(callName) {
				return substSegsFromNode(c, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
			}
			if shellBuiltinNoAllowlistSegment(callName) {
				return substSegsFromNode(c, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
			}
			if readBuiltinSkipsAllowlistSegment(callName) {
				if callExprArgsContainDisallowedExpansionForReadBuiltin(c.Args[1:]) {
					return nil, true
				}
				return substSegsFromNode(c, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
			}
			allowVar := argv0AllowsVarArgs != nil && argv0AllowsVarArgs(callName)
			if commandAllowsShellExpansionInArgsPastArgv0(callName) {
				allowVar = true
			}
			if !allowVar && callExprArgsContainDisallowedExpansionForStructured(c.Args[1:]) {
				return nil, true
			}
		}
		sg, ok := ctx.segFromStmt(st)
		if !ok {
			return nil, false
		}
		return []allowSeg{sg}, false
	case *syntax.TestClause:
		return substSegsFromNode(c, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.DeclClause:
		sg, ok := ctx.segFromStmt(st)
		if !ok {
			return nil, false
		}
		return []allowSeg{sg}, false
	case *syntax.LetClause, *syntax.ArithmCmd:
		return nil, false
	case *syntax.FuncDecl:
		if c.Body == nil {
			return nil, false
		}
		return stmtSegments(c.Body, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	case *syntax.CoprocClause, *syntax.TestDecl:
		return nil, true
	case *syntax.TimeClause:
		if c.Stmt == nil {
			return nil, false
		}
		return stmtSegments(c.Stmt, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	default:
		return nil, true
	}
}

// extractBashCScript returns the script argument after a literal -c or -lc when the command is bash or sh.
func extractBashCScript(ce *syntax.CallExpr) (string, bool) {
	s, _, ok := extractBashCScriptAndWord(ce)
	return s, ok
}

func extractBashCScriptAndWord(ce *syntax.CallExpr) (script string, word *syntax.Word, ok bool) {
	if ce == nil || len(ce.Args) < 3 {
		return "", nil, false
	}
	if !isBashOrSh(ce.Args[0].Lit()) {
		return "", nil, false
	}
	for i := 1; i < len(ce.Args)-1; i++ {
		lit := ce.Args[i].Lit()
		if lit != "-c" && lit != "-lc" {
			continue
		}
		w := ce.Args[i+1]
		s, ok := scriptFromWord(w)
		if !ok || s == "" {
			return "", nil, false
		}
		return s, w, true
	}
	return "", nil, false
}

// scriptContentBaseInRoot maps the inner -c script string to a byte offset in root where script[0] appears.
func scriptContentBaseInRoot(root string, w *syntax.Word, script string) (base int, ok bool) {
	if w == nil || script == "" {
		return 0, false
	}
	ws := int(w.Pos().Offset())
	we := int(w.End().Offset())
	if ws < 0 || we > len(root) || ws > we {
		return 0, false
	}
	if len(w.Parts) == 1 {
		if sq, ok := w.Parts[0].(*syntax.SglQuoted); ok && sq.Value == script {
			return ws + 1, true
		}
	}
	chunk := root[ws:we]
	if idx := strings.Index(chunk, script); idx >= 0 {
		return ws + idx, true
	}
	trim := strings.TrimSpace(chunk)
	if idx := strings.Index(trim, script); idx >= 0 {
		prefixLen := len(chunk) - len(strings.TrimLeft(chunk, " \t"))
		return ws + prefixLen + idx, true
	}
	return 0, false
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

// testOrBracketBuiltin is true for [ and test invoked as a simple command (common spelling of if-conditions).
func testOrBracketBuiltin(name string) bool {
	switch name {
	case "[", "test":
		return true
	}
	return strings.HasSuffix(name, "/[") || strings.HasSuffix(name, "/test")
}

// shellBuiltinNoAllowlistSegment reports builtins that are not argv[0] allowlist targets: they add no
// segment. Residual risk: arguments to ":" can still run expansions (e.g. "$(cmd)"); those appear as CmdSubst.
func shellBuiltinNoAllowlistSegment(name string) bool {
	switch argv0Base(name) {
	case ":", "continue", "break", "return":
		return true
	default:
		return false
	}
}

// readBuiltinSkipsAllowlistSegment is true for the read builtin: it adds no allowlist segment. Variable
// definitions are not executed as argv[0] binaries; use sites of those variables remain governed by
// structured opaque rules (e.g. kubectl -n "$ns"). If read's arguments contain disallowed expansions
// (unquoted $, command substitution in -p, etc.), [stmtSegments] still rejects.
func readBuiltinSkipsAllowlistSegment(name string) bool {
	return argv0Base(name) == "read"
}

// simpleCallCommandName returns the first literal word of a simple command (the "name" position),
// or "" if the name is not a single literal (e.g. variable or expansion).
func simpleCallCommandName(ce *syntax.CallExpr) string {
	if ce == nil || len(ce.Args) == 0 {
		return ""
	}
	return ce.Args[0].Lit()
}

func walkIfClause(ic *syntax.IfClause, ctx walkCtx, localFuncs map[string]struct{}, unwrapLeft int, argv0AllowsVarArgs func(string) bool) (segs []allowSeg, reject bool) {
	s1, r1 := walkStmtList(ic.Cond, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	s2, r2 := walkStmtList(ic.Then, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
	segs = append(append(segs, s1...), s2...)
	reject = r1 || r2
	if ic.Else != nil {
		s3, r3 := walkIfClause(ic.Else, ctx, localFuncs, unwrapLeft, argv0AllowsVarArgs)
		segs = append(segs, s3...)
		reject = reject || r3
	}
	return segs, reject
}
