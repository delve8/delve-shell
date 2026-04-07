package hil

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/benhoyt/goawk/lexer"
	"github.com/benhoyt/goawk/parser"
	"mvdan.cc/sh/v3/syntax"
)

// benignAwkReadOnly is true when the segment is a single invocable awk command (not gawk/mawk),
// uses no -f/--file/-e, parses as POSIX AWK via GoAWK, and the program has no file/pipe output,
// system(), or shell-backed getline.
//
// Residual risk: awk may still read files named in ARGV; that is out of scope for this check
// (same as cat/grep reading paths). GoAWK is used as a static analyzer only.
func benignAwkReadOnly(seg string) bool {
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
	if !ok || len(ce.Args) == 0 {
		return false
	}
	if !awkInvokerAllowed(ce.Args[0]) {
		return false
	}
	srcs, err := awkProgramSourcesFromCall(ce)
	if err != nil {
		return false
	}
	for _, src := range srcs {
		if !goawkProgramReadOnly(src) {
			return false
		}
	}
	return true
}

func awkInvokerAllowed(cmd0 *syntax.Word) bool {
	name := strings.TrimSpace(cmd0.Lit())
	if name == "" {
		return false
	}
	base := name
	if i := strings.LastIndex(name, "/"); i >= 0 {
		base = name[i+1:]
	}
	// Only the portable "awk" binary name; gawk/mawk/nawk stay behind HIL.
	return base == "awk"
}

var errAwkSource = errors.New("awk source extraction rejected")

func awkProgramSourcesFromCall(ce *syntax.CallExpr) ([][]byte, error) {
	if len(ce.Args) < 2 {
		return nil, errAwkSource
	}
	i := 1
	for i < len(ce.Args) {
		flag, ok := awkFlagWord(ce.Args[i])
		if !ok {
			return nil, errAwkSource
		}
		if flag == "" {
			return nil, errAwkSource
		}
		if flag == "-" || !strings.HasPrefix(flag, "-") {
			break
		}
		switch {
		case flag == "-f" || flag == "--file" || (strings.HasPrefix(flag, "-f") && len(flag) > 2):
			return nil, errAwkSource
		case flag == "-e" || flag == "--source" || flag == "--include":
			return nil, errAwkSource
		case flag == "-v":
			i += 2
			if i > len(ce.Args) {
				return nil, errAwkSource
			}
			continue
		case strings.HasPrefix(flag, "-v") && strings.Contains(flag, "="):
			i++
			continue
		case flag == "-F":
			i += 2
			if i > len(ce.Args) {
				return nil, errAwkSource
			}
			continue
		case strings.HasPrefix(flag, "-F"):
			i++
			continue
		default:
			// Reject non-portable flags (-W, --csv, -i, etc.).
			return nil, errAwkSource
		}
	}
	if i >= len(ce.Args) {
		return nil, errAwkSource
	}
	prog, err := awkSourceBytesFromWord(ce.Args[i])
	if err != nil {
		return nil, err
	}
	return [][]byte{prog}, nil
}

func awkFlagWord(w *syntax.Word) (string, bool) {
	if w == nil {
		return "", false
	}
	if lit, ok := wordSingleLitOrSglQuoted(w); ok {
		return lit, true
	}
	var b strings.Builder
	if err := syntax.NewPrinter().Print(&b, w); err != nil {
		return "", false
	}
	return strings.TrimSpace(b.String()), true
}

func wordSingleLitOrSglQuoted(w *syntax.Word) (string, bool) {
	if len(w.Parts) != 1 {
		return "", false
	}
	switch p := w.Parts[0].(type) {
	case *syntax.Lit:
		return p.Value, true
	case *syntax.SglQuoted:
		return p.Value, true
	default:
		return "", false
	}
}

func awkSourceBytesFromWord(w *syntax.Word) ([]byte, error) {
	if w == nil {
		return nil, errAwkSource
	}
	if len(w.Parts) == 1 {
		if sq, ok := w.Parts[0].(*syntax.SglQuoted); ok {
			return []byte(sq.Value), nil
		}
	}
	var b strings.Builder
	if err := syntax.NewPrinter().Print(&b, w); err != nil {
		return nil, errAwkSource
	}
	s := strings.TrimSpace(b.String())
	if len(s) >= 2 {
		switch {
		case s[0] == '\'' && s[len(s)-1] == '\'':
			return []byte(s[1 : len(s)-1]), nil
		case s[0] == '"':
			u, err := strconv.Unquote(s)
			if err == nil {
				return []byte(u), nil
			}
		}
	}
	if s == "" {
		return nil, errAwkSource
	}
	return []byte(s), nil
}

func goawkProgramReadOnly(src []byte) bool {
	if len(strings.TrimSpace(string(src))) == 0 {
		return false
	}
	prog, err := parser.ParseProgram(src, nil)
	if err != nil {
		return false
	}
	rv := reflect.ValueOf(prog).Elem()
	return walkGoawkProgram(rv)
}

func walkGoawkProgram(rv reflect.Value) bool {
	if !walkGoawkStmtBlocks(rv.FieldByName("Begin")) {
		return false
	}
	actions := rv.FieldByName("Actions")
	for i := 0; i < actions.Len(); i++ {
		act := derefPtrIface(actions.Index(i))
		if !act.IsValid() {
			continue
		}
		if !walkGoawkStmts(act.FieldByName("Stmts")) {
			return false
		}
	}
	if !walkGoawkStmtBlocks(rv.FieldByName("End")) {
		return false
	}
	funcs := rv.FieldByName("Functions")
	for i := 0; i < funcs.Len(); i++ {
		fn := derefPtrIface(funcs.Index(i))
		if !fn.IsValid() {
			continue
		}
		if !walkGoawkStmts(fn.FieldByName("Body")) {
			return false
		}
	}
	return true
}

// walkGoawkStmtBlocks walks []Stmts (BEGIN/END may have multiple blocks).
func walkGoawkStmtBlocks(blocks reflect.Value) bool {
	if !blocks.IsValid() {
		return true
	}
	for blocks.Kind() == reflect.Ptr && !blocks.IsNil() {
		blocks = blocks.Elem()
	}
	if !blocks.IsValid() || blocks.Kind() != reflect.Slice {
		return true
	}
	for i := 0; i < blocks.Len(); i++ {
		if !walkGoawkStmts(blocks.Index(i)) {
			return false
		}
	}
	return true
}

func walkGoawkStmts(ss reflect.Value) bool {
	if !ss.IsValid() {
		return true
	}
	for ss.Kind() == reflect.Ptr && !ss.IsNil() {
		ss = ss.Elem()
	}
	if !ss.IsValid() || ss.Kind() != reflect.Slice {
		return true
	}
	for i := 0; i < ss.Len(); i++ {
		si := ss.Index(i)
		for si.Kind() == reflect.Interface && !si.IsNil() {
			si = si.Elem()
		}
		if !si.IsValid() {
			continue
		}
		if !walkGoawkStmt(si) {
			return false
		}
	}
	return true
}

func walkGoawkStmt(sv reflect.Value) bool {
	sv = derefPtrIface(sv)
	if !sv.IsValid() {
		return true
	}
	switch sv.Type().Name() {
	case "PrintStmt", "PrintfStmt":
		redir := sv.FieldByName("Redirect").Interface().(lexer.Token)
		if redir != lexer.ILLEGAL {
			return false
		}
		dest := sv.FieldByName("Dest")
		if dest.IsValid() && !dest.IsNil() {
			return false
		}
		return walkGoawkExprSlice(sv.FieldByName("Args"))
	case "ExprStmt":
		return walkGoawkExpr(derefPtrIface(sv.FieldByName("Expr")))
	case "IfStmt":
		if !walkGoawkExpr(derefPtrIface(sv.FieldByName("Cond"))) {
			return false
		}
		if !walkGoawkStmts(sv.FieldByName("Body")) {
			return false
		}
		return walkGoawkStmts(sv.FieldByName("Else"))
	case "ForStmt":
		if pre := sv.FieldByName("Pre"); pre.IsValid() && !pre.IsNil() {
			if !walkGoawkStmt(derefPtrIface(pre)) {
				return false
			}
		}
		if !walkGoawkExpr(derefPtrIface(sv.FieldByName("Cond"))) {
			return false
		}
		if post := sv.FieldByName("Post"); post.IsValid() && !post.IsNil() {
			if !walkGoawkStmt(derefPtrIface(post)) {
				return false
			}
		}
		return walkGoawkStmts(sv.FieldByName("Body"))
	case "ForInStmt":
		return walkGoawkStmts(sv.FieldByName("Body"))
	case "WhileStmt":
		if !walkGoawkExpr(derefPtrIface(sv.FieldByName("Cond"))) {
			return false
		}
		return walkGoawkStmts(sv.FieldByName("Body"))
	case "DoWhileStmt":
		if !walkGoawkStmts(sv.FieldByName("Body")) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(sv.FieldByName("Cond")))
	case "BreakStmt", "ContinueStmt", "NextStmt", "NextfileStmt":
		return true
	case "ExitStmt":
		return walkGoawkExpr(derefPtrIface(sv.FieldByName("Status")))
	case "DeleteStmt":
		return walkGoawkExprSlice(sv.FieldByName("Index"))
	case "ReturnStmt":
		return walkGoawkExpr(derefPtrIface(sv.FieldByName("Value")))
	case "BlockStmt":
		return walkGoawkStmts(sv.FieldByName("Body"))
	default:
		return false
	}
}

func walkGoawkExprSlice(ev reflect.Value) bool {
	if !ev.IsValid() || ev.Kind() != reflect.Slice {
		return true
	}
	for i := 0; i < ev.Len(); i++ {
		if !walkGoawkExpr(derefPtrIface(ev.Index(i))) {
			return false
		}
	}
	return true
}

func walkGoawkExpr(ev reflect.Value) bool {
	ev = derefPtrIface(ev)
	if !ev.IsValid() {
		return true
	}
	switch ev.Type().Name() {
	case "FieldExpr":
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Index")))
	case "NamedFieldExpr":
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Field")))
	case "UnaryExpr":
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Value")))
	case "BinaryExpr":
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("Left"))) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Right")))
	case "InExpr":
		return walkGoawkExprSlice(ev.FieldByName("Index"))
	case "CondExpr":
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("Cond"))) {
			return false
		}
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("True"))) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("False")))
	case "NumExpr", "StrExpr", "RegExpr", "VarExpr":
		return true
	case "IndexExpr":
		return walkGoawkExprSlice(ev.FieldByName("Index"))
	case "AssignExpr":
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("Left"))) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Right")))
	case "AugAssignExpr":
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("Left"))) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Right")))
	case "IncrExpr":
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Expr")))
	case "CallExpr":
		tok := ev.FieldByName("Func").Interface().(lexer.Token)
		if tok == lexer.F_SYSTEM {
			return false
		}
		return walkGoawkExprSlice(ev.FieldByName("Args"))
	case "UserCallExpr":
		return walkGoawkExprSlice(ev.FieldByName("Args"))
	case "MultiExpr":
		return walkGoawkExprSlice(ev.FieldByName("Exprs"))
	case "GetlineExpr":
		cmd := ev.FieldByName("Command")
		if cmd.IsValid() && !cmd.IsNil() {
			return false
		}
		if !walkGoawkExpr(derefPtrIface(ev.FieldByName("Target"))) {
			return false
		}
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("File")))
	case "GroupingExpr":
		return walkGoawkExpr(derefPtrIface(ev.FieldByName("Expr")))
	default:
		return false
	}
}

func derefPtrIface(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v
}
