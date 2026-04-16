package hil

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

const maxStaticValueVariants = 32

type staticValueEnv map[string][]string

type allowSegmentInfo struct {
	span allowSeg
	env  staticValueEnv
}

func cloneStaticValueEnv(env staticValueEnv) staticValueEnv {
	if len(env) == 0 {
		return nil
	}
	out := make(staticValueEnv, len(env))
	for name, values := range env {
		out[name] = append([]string(nil), values...)
	}
	return out
}

func withStaticValues(env staticValueEnv, name string, values []string) staticValueEnv {
	next := cloneStaticValueEnv(env)
	if len(values) == 0 {
		delete(next, name)
		return next
	}
	if next == nil {
		next = make(staticValueEnv, 1)
	}
	next[name] = append([]string(nil), values...)
	return next
}

func withoutStaticValue(env staticValueEnv, name string) staticValueEnv {
	if len(env) == 0 {
		return nil
	}
	if _, ok := env[name]; !ok {
		return env
	}
	next := cloneStaticValueEnv(env)
	delete(next, name)
	return next
}

func simpleParamName(w *syntax.Word) (string, bool) {
	if w == nil || len(w.Parts) != 1 {
		return "", false
	}
	dq, ok := w.Parts[0].(*syntax.DblQuoted)
	if !ok || len(dq.Parts) != 1 {
		return "", false
	}
	pe, ok := dq.Parts[0].(*syntax.ParamExp)
	if !ok || !isSimpleParamExp(pe) || pe.Param == nil {
		return "", false
	}
	return pe.Param.Value, pe.Param.Value != ""
}

func staticAssignmentValue(a *syntax.Assign) (string, bool) {
	if a == nil || a.Naked || a.Append || a.Index != nil || a.Array != nil {
		return "", false
	}
	if a.Value == nil {
		return "", true
	}
	if WordContainsShellExpansion(a.Value) || WordContainsExtGlob(a.Value) {
		return "", false
	}
	var b strings.Builder
	for _, p := range a.Value.Parts {
		if !appendWordPartStatic(&b, p) {
			return "", false
		}
	}
	return b.String(), true
}

func staticAssignmentValues(env staticValueEnv, a *syntax.Assign) ([]string, bool) {
	if a == nil || a.Naked || a.Append || a.Index != nil || a.Array != nil {
		return nil, false
	}
	if a.Value == nil {
		return []string{""}, true
	}
	return staticValuesFromWord(a.Value, env)
}

func staticValuesFromWord(w *syntax.Word, env staticValueEnv) ([]string, bool) {
	if name, ok := simpleParamName(w); ok {
		values := env[name]
		if len(values) == 0 {
			return nil, false
		}
		return append([]string(nil), values...), true
	}
	if w == nil || WordContainsShellExpansion(w) || WordContainsExtGlob(w) {
		return nil, false
	}
	var b strings.Builder
	for _, p := range w.Parts {
		if !appendWordPartStatic(&b, p) {
			return nil, false
		}
	}
	return []string{b.String()}, true
}

func staticValuesFromWordItems(items []*syntax.Word, env staticValueEnv) ([]string, bool) {
	if len(items) == 0 {
		return nil, true
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		values, ok := staticValuesFromWord(item, env)
		if !ok {
			return nil, false
		}
		for _, v := range values {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
			if len(out) > maxStaticValueVariants {
				return nil, false
			}
		}
	}
	return out, true
}

func updateEnvForAssign(env staticValueEnv, a *syntax.Assign) staticValueEnv {
	if a == nil || a.Name == nil || a.Name.Value == "" {
		return env
	}
	if values, ok := staticAssignmentValues(env, a); ok {
		return withStaticValues(env, a.Name.Value, values)
	}
	return withoutStaticValue(env, a.Name.Value)
}

func updateEnvForAssigns(env staticValueEnv, assigns []*syntax.Assign) staticValueEnv {
	next := env
	for _, a := range assigns {
		next = updateEnvForAssign(next, a)
	}
	return next
}

func narrowEnvFromCondition(stmts []*syntax.Stmt, env staticValueEnv) staticValueEnv {
	name, values, ok := conditionEqualsStaticValues(stmts, env)
	if !ok {
		return env
	}
	return withStaticValues(env, name, values)
}

func conditionEqualsStaticValues(stmts []*syntax.Stmt, env staticValueEnv) (string, []string, bool) {
	if len(stmts) != 1 || stmts[0] == nil || stmts[0].Cmd == nil {
		return "", nil, false
	}
	switch c := stmts[0].Cmd.(type) {
	case *syntax.CallExpr:
		return callExprEqualsStaticValues(c, env)
	case *syntax.TestClause:
		return testExprEqualsStaticValues(c.X, env)
	default:
		return "", nil, false
	}
}

func callExprEqualsStaticValues(c *syntax.CallExpr, env staticValueEnv) (string, []string, bool) {
	if c == nil || len(c.Args) < 4 {
		return "", nil, false
	}
	name := argv0Base(simpleCallCommandName(c))
	args := c.Args
	switch name {
	case "[":
		if len(args) != 5 || args[4] == nil || args[4].Lit() != "]" {
			return "", nil, false
		}
		return wordsEqualityStaticValues(args[1], args[2], args[3], env)
	case "test":
		if len(args) != 4 {
			return "", nil, false
		}
		return wordsEqualityStaticValues(args[1], args[2], args[3], env)
	default:
		return "", nil, false
	}
}

func testExprEqualsStaticValues(expr syntax.TestExpr, env staticValueEnv) (string, []string, bool) {
	bt, ok := expr.(*syntax.BinaryTest)
	if !ok || (bt.Op != syntax.TsMatchShort && bt.Op != syntax.TsMatch) {
		return "", nil, false
	}
	return testExprWordsEqualityStaticValues(bt.X, bt.Y, env)
}

func testExprWordsEqualityStaticValues(left syntax.TestExpr, right syntax.TestExpr, env staticValueEnv) (string, []string, bool) {
	lw, lok := left.(*syntax.Word)
	rw, rok := right.(*syntax.Word)
	if !lok || !rok {
		return "", nil, false
	}
	return equalityStaticValues(lw, rw, env)
}

func wordsEqualityStaticValues(left *syntax.Word, op *syntax.Word, right *syntax.Word, env staticValueEnv) (string, []string, bool) {
	if op == nil {
		return "", nil, false
	}
	switch op.Lit() {
	case "=", "==":
		return equalityStaticValues(left, right, env)
	default:
		return "", nil, false
	}
}

func equalityStaticValues(left *syntax.Word, right *syntax.Word, env staticValueEnv) (string, []string, bool) {
	if name, ok := simpleParamName(left); ok {
		if values, ok := exactStaticValuesFromWord(right, env); ok {
			return name, values, true
		}
	}
	if name, ok := simpleParamName(right); ok {
		if values, ok := exactStaticValuesFromWord(left, env); ok {
			return name, values, true
		}
	}
	return "", nil, false
}

func exactStaticValuesFromWord(w *syntax.Word, env staticValueEnv) ([]string, bool) {
	values, ok := staticValuesFromWord(w, env)
	if !ok || len(values) == 0 {
		return nil, false
	}
	for _, v := range values {
		if stringContainsShellPatternMeta(v) {
			return nil, false
		}
	}
	return values, true
}

func staticValuesFromCasePatterns(patterns []*syntax.Word, env staticValueEnv) ([]string, bool) {
	values, ok := staticValuesFromWordItems(patterns, env)
	if !ok || len(values) == 0 {
		return nil, false
	}
	for _, v := range values {
		if stringContainsShellPatternMeta(v) {
			return nil, false
		}
	}
	return values, true
}

func stringContainsShellPatternMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func updateEnvForReadBuiltin(env staticValueEnv, ce *syntax.CallExpr) staticValueEnv {
	if ce == nil || len(ce.Args) < 2 {
		return env
	}
	next := env
	for i := 1; i < len(ce.Args); i++ {
		w := ce.Args[i]
		if w == nil || len(w.Parts) != 1 {
			continue
		}
		lit, ok := w.Parts[0].(*syntax.Lit)
		if !ok || lit.Value == "" || strings.HasPrefix(lit.Value, "-") {
			continue
		}
		next = withoutStaticValue(next, lit.Value)
	}
	return next
}

func staticOrResolvedSimpleCommandArgVariants(seg string, env staticValueEnv) ([][]readOnlyCLIArg, bool) {
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
	variants := [][]readOnlyCLIArg{make([]readOnlyCLIArg, 0, len(ce.Args))}
	for i, w := range ce.Args {
		if w == nil {
			return nil, false
		}
		if i == 0 {
			lit := w.Lit()
			if lit == "" {
				return nil, false
			}
			for j := range variants {
				variants[j] = append(variants[j], readOnlyCLIArg{lit: lit})
			}
			continue
		}
		if WordContainsExtGlob(w) {
			return nil, false
		}
		if flagToken, ok := wordIsFlagWithDoubleQuotedCmdSubstValue(w); ok {
			for j := range variants {
				variants[j] = append(variants[j], readOnlyCLIArg{flagToken: flagToken, cmdSubst: true})
			}
			continue
		}
		if name, ok := simpleParamName(w); ok {
			if values := env[name]; len(values) > 0 {
				next := make([][]readOnlyCLIArg, 0, len(variants)*len(values))
				for _, variant := range variants {
					for _, value := range values {
						cp := append([]readOnlyCLIArg(nil), variant...)
						cp = append(cp, readOnlyCLIArg{lit: value})
						next = append(next, cp)
						if len(next) > maxStaticValueVariants {
							return nil, false
						}
					}
				}
				variants = next
				continue
			}
			for j := range variants {
				variants[j] = append(variants[j], readOnlyCLIArg{opaque: true})
			}
			continue
		}
		if wordIsDoubleQuotedCmdSubstOnly(w) {
			for j := range variants {
				variants[j] = append(variants[j], readOnlyCLIArg{cmdSubst: true})
			}
			continue
		}
		if wordContainsDisallowedShellExpansionForStructured(w) {
			return nil, false
		}
		s, ok := wordToStaticString(w)
		if !ok {
			return nil, false
		}
		for j := range variants {
			variants[j] = append(variants[j], readOnlyCLIArg{lit: s})
		}
	}
	return variants, true
}
