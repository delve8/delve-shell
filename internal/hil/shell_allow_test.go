package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestCommandAllowsAutoApprove_ParseFailRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if w.CommandAllowsAutoApprove("echo \"unclosed") {
		t.Fatal("invalid shell should not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_EgrepQuotedPipe(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	cmd := "kubectl get pods -n default 2>/dev/null | egrep 'foo|bar|baz' || true"
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want auto-approve for quoted | inside egrep: %q", cmd)
	}
}

func TestCommandAllowsAutoApprove_FuncDeclCallSkippedWhenNameIsLocal(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	// Body is allowlisted; call to f is not matched as its own segment (same-script function).
	if !w.CommandAllowsAutoApprove("f() { kubectl get pods -n default; }; f") {
		t.Fatal("want auto-approve: body checked, local function name not required on allowlist")
	}
}

func TestCommandAllowsAutoApprove_UndefinedFuncCallStillChecked(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if w.CommandAllowsAutoApprove("f") {
		t.Fatal("undefined f should not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_FuncDeclBodyOnlyWhenAllowlisted(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if !w.CommandAllowsAutoApprove("f() { kubectl get ns; }") {
		t.Fatal("function body commands should be checked like normal statements")
	}
	if w.CommandAllowsAutoApprove("f() { rm -rf /; }") {
		t.Fatal("disallowed body command should block auto-approve")
	}
}

func TestCommandAllowsAutoApprove_CmdSubstInnerMustBeAllowed(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if w.CommandAllowsAutoApprove("echo $(rm -rf /)") {
		t.Fatal("inner rm must block auto-approve")
	}
}

func TestCommandAllowsAutoApprove_EmptyArithmNoSegments(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if !w.CommandAllowsAutoApprove("((1+1))") {
		t.Fatal("pure arithmetic with no external commands may auto-approve")
	}
}

func TestCommandAllowsAutoApprove_BashCUnwrapOneLevel(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	if !w.CommandAllowsAutoApprove("bash -c 'kubectl get ns'") {
		t.Fatal("want unwrap one bash -c and allowlist inner kubectl")
	}
	if !w.CommandAllowsAutoApprove("bash -lc 'set -e; kubectl get ns'") {
		t.Fatal("want bash -lc unwrap with set and kubectl")
	}
	if w.CommandAllowsAutoApprove("bash -c 'bash -c \"kubectl get ns\"'") {
		t.Fatal("nested bash -c must not unwrap twice (outer bash segment would need allowlist)")
	}
}

func TestCommandAllowsAutoApprove_JqPipeline(t *testing.T) {
	w := NewAllowlist(config.DefaultAllowlistEntries())
	cmd := `bash -c 'kubectl get pods -A -o json | jq -r ".items | length" | head -n 1'`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want jq + head in unwrapped -c: %q", cmd)
	}
	if w.CommandAllowsAutoApprove(`bash -c 'jq -f /tmp/x.jq .'`) {
		t.Fatal("jq -f should not auto-approve")
	}
}
