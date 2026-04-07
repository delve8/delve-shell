package hil

import (
	"testing"

	"mvdan.cc/sh/v3/syntax"
)

// callExprArg returns ce.Args[idx] from a single-line simple command (e.g. echo …, awk …).
func callExprArg(t *testing.T, line string, idx int) *syntax.Word {
	t.Helper()
	f, err := parseShell(line)
	if err != nil {
		t.Fatalf("parse %q: %v", line, err)
	}
	if len(f.Stmts) != 1 {
		t.Fatalf("want 1 stmt, got %d in %q", len(f.Stmts), line)
	}
	st := f.Stmts[0]
	if st == nil || st.Cmd == nil {
		t.Fatalf("empty stmt: %q", line)
	}
	ce, ok := st.Cmd.(*syntax.CallExpr)
	if !ok {
		t.Fatalf("want CallExpr: %q", line)
	}
	if idx < 0 || idx >= len(ce.Args) {
		t.Fatalf("arg idx %d out of range (n=%d) in %q", idx, len(ce.Args), line)
	}
	return ce.Args[idx]
}

func TestWordContainsShellExpansion_quotingAndEscapes(t *testing.T) {
	tests := []struct {
		line string
		idx  int
		want bool
	}{
		// Single-quoted: $ is not shell expansion at parse level.
		{`echo '$HOME'`, 1, false},
		{`echo '${HOME}'`, 1, false},
		{`echo '$1'`, 1, false},
		// Double-quoted: parameter expansion.
		{`echo "$HOME"`, 1, true},
		{`echo "${HOME}"`, 1, true},
		{`echo "$1"`, 1, true},
		// Escaped $ inside double quotes (often literal for later tools).
		{`echo "\$HOME"`, 1, false},
		{`awk "{print \$1}"`, 1, false},
		// Unescaped $1 in double-quoted awk program: ParamExp in shell AST.
		{`awk "{print $1}"`, 1, true},
		// Command / arithmetic substitution.
		{`echo $(date)`, 1, true},
		{`echo $((1+1))`, 1, true},
		// Unquoted (still one Word if no spaces inside).
		{`echo $HOME`, 1, true},
		// Literal braces without expansion in unquoted lit-only word.
		{`echo hello`, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			w := callExprArg(t, tt.line, tt.idx)
			got := WordContainsShellExpansion(w)
			if got != tt.want {
				t.Fatalf("WordContainsShellExpansion(%q arg %d) = %v, want %v", tt.line, tt.idx, got, tt.want)
			}
		})
	}
}

func TestCallExprArgsContainShellExpansion(t *testing.T) {
	f, err := parseShell(`kubectl get ns -n default`)
	if err != nil {
		t.Fatal(err)
	}
	ce := f.Stmts[0].Cmd.(*syntax.CallExpr)
	if callExprArgsContainShellExpansion(ce.Args[1:]) {
		t.Fatal("literal kubectl args should not contain expansion")
	}
	f2, err := parseShell(`kubectl get ns -n "${NS}"`)
	if err != nil {
		t.Fatal(err)
	}
	ce2 := f2.Stmts[0].Cmd.(*syntax.CallExpr)
	if !callExprArgsContainShellExpansion(ce2.Args[1:]) {
		t.Fatal("kubectl with ${NS} should report expansion in args")
	}
}

func TestCommandAllowsShellExpansionInArgsPastArgv0(t *testing.T) {
	if !commandAllowsShellExpansionInArgsPastArgv0("awk") {
		t.Fatal("awk should allow expansion scan skip")
	}
	if !commandAllowsShellExpansionInArgsPastArgv0("/usr/local/bin/awk") {
		t.Fatal("basename awk should match")
	}
	if !commandAllowsShellExpansionInArgsPastArgv0("/usr/bin/gawk") {
		t.Fatal("basename gawk should match")
	}
	if commandAllowsShellExpansionInArgsPastArgv0("kubectl") {
		t.Fatal("kubectl should not skip expansion scan")
	}
}

func TestWordContainsExtGlob(t *testing.T) {
	tests := []struct {
		line string
		idx  int
		want bool
	}{
		{`echo 'foo?(bar)'`, 1, false},
		{`echo foo?(bar)`, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			w := callExprArg(t, tt.line, tt.idx)
			got := WordContainsExtGlob(w)
			if got != tt.want {
				t.Fatalf("WordContainsExtGlob(arg %d of %q) = %v, want %v", tt.idx, tt.line, got, tt.want)
			}
		})
	}
}
