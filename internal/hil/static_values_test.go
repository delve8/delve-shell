package hil

import (
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

func mustParseFile(t *testing.T, command string) *syntax.File {
	t.Helper()
	f, err := parseShell(command)
	if err != nil {
		t.Fatalf("parseShell(%q): %v", command, err)
	}
	return f
}

func mustFirstAssign(t *testing.T, command string) *syntax.Assign {
	t.Helper()
	f := mustParseFile(t, command)
	if len(f.Stmts) != 1 || f.Stmts[0] == nil || f.Stmts[0].Cmd == nil {
		t.Fatalf("expected one stmt in %q", command)
	}
	ce, ok := f.Stmts[0].Cmd.(*syntax.CallExpr)
	if !ok || len(ce.Assigns) == 0 {
		t.Fatalf("expected assignment-only call in %q", command)
	}
	return ce.Assigns[0]
}

func mustFirstWordArg(t *testing.T, command string, index int) *syntax.Word {
	t.Helper()
	f := mustParseFile(t, command)
	if len(f.Stmts) != 1 || f.Stmts[0] == nil || f.Stmts[0].Cmd == nil {
		t.Fatalf("expected one stmt in %q", command)
	}
	ce, ok := f.Stmts[0].Cmd.(*syntax.CallExpr)
	if !ok || index >= len(ce.Args) {
		t.Fatalf("expected arg %d in %q", index, command)
	}
	return ce.Args[index]
}

func mustCasePatterns(t *testing.T, command string) []*syntax.Word {
	t.Helper()
	f := mustParseFile(t, command)
	if len(f.Stmts) != 1 || f.Stmts[0] == nil || f.Stmts[0].Cmd == nil {
		t.Fatalf("expected one stmt in %q", command)
	}
	cc, ok := f.Stmts[0].Cmd.(*syntax.CaseClause)
	if !ok || len(cc.Items) == 0 {
		t.Fatalf("expected case clause in %q", command)
	}
	return cc.Items[0].Patterns
}

func TestStaticAssignmentValues(t *testing.T) {
	env := staticValueEnv{"verb": []string{"get"}}
	tests := []struct {
		name string
		cmd  string
		want []string
		ok   bool
	}{
		{name: "literal", cmd: `verb=get`, want: []string{"get"}, ok: true},
		{name: "alias", cmd: `alias="$verb"`, want: []string{"get"}, ok: true},
		{name: "empty", cmd: `verb=`, want: []string{""}, ok: true},
		{name: "cmd subst", cmd: `verb=$(printf get)`, ok: false},
		{name: "append", cmd: `verb+=x`, ok: false},
		{name: "index", cmd: `verb[0]=x`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := staticAssignmentValues(env, mustFirstAssign(t, tt.cmd))
			if ok != tt.ok {
				t.Fatalf("staticAssignmentValues(%q) ok=%v want %v", tt.cmd, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("staticAssignmentValues(%q)=%v want %v", tt.cmd, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("staticAssignmentValues(%q)=%v want %v", tt.cmd, got, tt.want)
				}
			}
		})
	}
}

func TestStaticValuesFromWord(t *testing.T) {
	env := staticValueEnv{"verb": []string{"get", "describe"}}
	tests := []struct {
		name string
		cmd  string
		arg  int
		want []string
		ok   bool
	}{
		{name: "literal", cmd: `echo get`, arg: 1, want: []string{"get"}, ok: true},
		{name: "quoted simple param", cmd: `echo "$verb"`, arg: 1, want: []string{"get", "describe"}, ok: true},
		{name: "unknown simple param", cmd: `echo "$missing"`, arg: 1, ok: false},
		{name: "command substitution", cmd: `echo "$(printf get)"`, arg: 1, ok: false},
		{name: "extglob", cmd: `echo ?(get)`, arg: 1, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := staticValuesFromWord(mustFirstWordArg(t, tt.cmd, tt.arg), env)
			if ok != tt.ok {
				t.Fatalf("staticValuesFromWord(%q) ok=%v want %v", tt.cmd, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("staticValuesFromWord(%q)=%v want %v", tt.cmd, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("staticValuesFromWord(%q)=%v want %v", tt.cmd, got, tt.want)
				}
			}
		})
	}
}

func TestConditionEqualsStaticValues(t *testing.T) {
	env := staticValueEnv{"other": []string{"get"}}
	tests := []struct {
		name     string
		command  string
		wantName string
		wantVals []string
		wantOK   bool
	}{
		{name: "posix test", command: `[ "$verb" = get ]`, wantName: "verb", wantVals: []string{"get"}, wantOK: true},
		{name: "test builtin reverse", command: `test describe = "$verb"`, wantName: "verb", wantVals: []string{"describe"}, wantOK: true},
		{name: "double bracket alias", command: `[[ "$verb" == "$other" ]]`, wantName: "verb", wantVals: []string{"get"}, wantOK: true},
		{name: "pattern", command: `[[ "$verb" == get* ]]`, wantOK: false},
		{name: "not equals", command: `[ "$verb" != get ]`, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := mustParseFile(t, tt.command)
			name, vals, ok := conditionEqualsStaticValues(f.Stmts, env)
			if ok != tt.wantOK {
				t.Fatalf("conditionEqualsStaticValues(%q) ok=%v want %v", tt.command, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if name != tt.wantName {
				t.Fatalf("conditionEqualsStaticValues(%q) name=%q want %q", tt.command, name, tt.wantName)
			}
			if len(vals) != len(tt.wantVals) {
				t.Fatalf("conditionEqualsStaticValues(%q) vals=%v want %v", tt.command, vals, tt.wantVals)
			}
			for i := range vals {
				if vals[i] != tt.wantVals[i] {
					t.Fatalf("conditionEqualsStaticValues(%q) vals=%v want %v", tt.command, vals, tt.wantVals)
				}
			}
		})
	}
}

func TestStaticValuesFromCasePatterns(t *testing.T) {
	env := staticValueEnv{"verb": []string{"get"}}
	tests := []struct {
		name string
		cmd  string
		want []string
		ok   bool
	}{
		{name: "literals", cmd: `case "$verb" in get|describe) : ;; esac`, want: []string{"get", "describe"}, ok: true},
		{name: "dynamic exact", cmd: `case "$verb" in "$verb") : ;; esac`, want: []string{"get"}, ok: true},
		{name: "glob", cmd: `case "$verb" in get*) : ;; esac`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := staticValuesFromCasePatterns(mustCasePatterns(t, tt.cmd), env)
			if ok != tt.ok {
				t.Fatalf("staticValuesFromCasePatterns(%q) ok=%v want %v", tt.cmd, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("staticValuesFromCasePatterns(%q)=%v want %v", tt.cmd, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("staticValuesFromCasePatterns(%q)=%v want %v", tt.cmd, got, tt.want)
				}
			}
		})
	}
}

func TestStaticOrResolvedSimpleCommandArgVariants(t *testing.T) {
	env := staticValueEnv{"verb": []string{"get", "describe"}}
	variants, ok := staticOrResolvedSimpleCommandArgVariants(`kubectl "$verb" pods`, env)
	if !ok {
		t.Fatal("expected variants for inferred simple param")
	}
	if len(variants) != 2 {
		t.Fatalf("variant count=%d want 2", len(variants))
	}
	got := []string{variants[0][1].lit, variants[1][1].lit}
	if got[0] != "get" || got[1] != "describe" {
		t.Fatalf("variant lits=%v want [get describe]", got)
	}

	opaqueVariants, ok := staticOrResolvedSimpleCommandArgVariants(`kubectl "$missing" pods`, nil)
	if !ok || len(opaqueVariants) != 1 || !opaqueVariants[0][1].opaque {
		t.Fatalf("expected opaque variant for unknown simple param, got=%+v ok=%v", opaqueVariants, ok)
	}
}

func TestStaticValuesFromWordItems_LimitStaysConservative(t *testing.T) {
	env := staticValueEnv{
		"verb": {
			"v01", "v02", "v03", "v04", "v05", "v06", "v07", "v08", "v09", "v10", "v11",
			"v12", "v13", "v14", "v15", "v16", "v17", "v18", "v19", "v20", "v21", "v22",
			"v23", "v24", "v25", "v26", "v27", "v28", "v29", "v30", "v31", "v32", "v33",
		},
	}
	f := mustParseFile(t, `for item in "$verb"; do :; done`)
	fc, ok := f.Stmts[0].Cmd.(*syntax.ForClause)
	if !ok {
		t.Fatal("expected for clause")
	}
	wi, ok := fc.Loop.(*syntax.WordIter)
	if !ok {
		t.Fatal("expected word iterator")
	}
	if _, ok := staticValuesFromWordItems(wi.Items, env); ok {
		t.Fatal("expected inferred value limit to fail closed")
	}
}
