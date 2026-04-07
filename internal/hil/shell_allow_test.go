package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestCommandAllowsAutoApprove_ParseFailRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove("echo \"unclosed") {
		t.Fatal("invalid shell should not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_EgrepQuotedPipe(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := "kubectl get pods -n default 2>/dev/null | egrep 'foo|bar|baz' || true"
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want auto-approve for quoted | inside egrep: %q", cmd)
	}
}

func TestCommandAllowsAutoApprove_FuncDeclCallSkippedWhenNameIsLocal(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	// Body is allowlisted; call to f is not matched as its own segment (same-script function).
	if !w.CommandAllowsAutoApprove("f() { kubectl get pods -n default; }; f") {
		t.Fatal("want auto-approve: body checked, local function name not required on allowlist")
	}
}

func TestCommandAllowsAutoApprove_UndefinedFuncCallStillChecked(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove("f") {
		t.Fatal("undefined f should not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_FuncDeclBodyOnlyWhenAllowlisted(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove("f() { kubectl get ns; }") {
		t.Fatal("function body commands should be checked like normal statements")
	}
	if w.CommandAllowsAutoApprove("f() { rm -rf /; }") {
		t.Fatal("disallowed body command should block auto-approve")
	}
}

func TestCommandAllowsAutoApprove_CmdSubstInnerMustBeAllowed(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove("echo $(rm -rf /)") {
		t.Fatal("inner rm must block auto-approve")
	}
}

func TestCommandAllowsAutoApprove_EmptyArithmNoSegments(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove("((1+1))") {
		t.Fatal("pure arithmetic with no external commands may auto-approve")
	}
}

func TestCommandAllowsAutoApprove_BashCUnwrapOneLevel(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
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
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := `bash -c 'kubectl get pods -A -o json | jq -r ".items | length" | head -n 1'`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want jq + head in unwrapped -c: %q", cmd)
	}
	if w.CommandAllowsAutoApprove(`bash -c 'jq -f /tmp/x.jq .'`) {
		t.Fatal("jq -f should not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_AwkInPipeline(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := `kubectl get pods -A --field-selector=status.phase=Pending -o wide 2>/dev/null | awk 'NR==1 || NR<=15{print}' || true`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want auto-approve for read-only awk filter: %q", cmd)
	}
}

func TestCommandAllowsAutoApprove_AwkDoubleQuotedFieldRefNotRejectedAsShellVar(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	// Double-quoted awk program: $1 is shell ParamExp; awk must be permissiveArgv0 so collectAllowlistSegments does not reject.
	if !w.CommandAllowsAutoApprove(`awk "{print $1}" </dev/null`) {
		t.Fatal("want auto-approve: $1 is awk field ref inside double quotes, not a kubectl-style expansion risk")
	}
}

func TestCommandAllowsAutoApprove_GawkNotBenignAwk(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`gawk '{print NR}'`) {
		t.Fatal("gawk should not use benign awk path")
	}
}

func TestCommandAllowsAutoApprove_AwkPrintRedirectRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`awk '{print > "/tmp/x"}'`) {
		t.Fatal("awk print redirect must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_AwkSystemRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`awk 'BEGIN{system("date")}'`) {
		t.Fatal("awk system() must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_AwkFromFileRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`awk -f /tmp/x.awk`) {
		t.Fatal("awk -f must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_TestClauseAndColonBuiltin(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`bash -lc 'if [ "${pending:-0}" != "0" ]; then echo ok; fi'`) {
		t.Fatal("want auto-approve: [ / test builtin and echo allowlisted")
	}
	if !w.CommandAllowsAutoApprove(`bash -lc 'if kubectl get ns 2>/dev/null; then :; else echo x; fi'`) {
		t.Fatal("want auto-approve: colon builtin adds no segment")
	}
}

func TestCommandAllowsAutoApprove_TestClauseCmdSubstStillChecked(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`bash -lc '[ "$(rm -rf /)" = x ]'`) {
		t.Fatal("command substitution inside TestClause must still be allowlisted")
	}
}

func TestCommandAllowsAutoApprove_DynamicArgv0RequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`bash -lc '$(echo true)'`) {
		t.Fatal("dynamic argv[0] must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_KubectlVarArgRequiresApproval(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`bash -lc 'kubectl get ns -n "${NS}"'`) {
		t.Fatal("kubectl with expansion in args must not auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`bash -lc 'kubectl get ns -n default'`) {
		t.Fatal("kubectl literal args should still auto-approve when allowlisted")
	}
}

func TestCommandAllowsAutoApprove_EchoVarArgStillAutoApprove(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`bash -lc 'echo "$PATH"'`) {
		t.Fatal("echo is read-only: expansions in args may auto-approve")
	}
}

func TestCommandAllowsAutoApprove_OneLinePendingPodsEventsScript(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	// Single-line -lc script (no newlines inside quotes); mirrors user pipeline without /tmp writes.
	cmd := `bash -lc 'set -euo pipefail; printf "Pending pods: "; kubectl get pods -A --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l | awk "{print \$1}"; printf "\nRecent scheduling-related events:\n"; kubectl get events -A --sort-by=.lastTimestamp 2>/dev/null | egrep "FailedScheduling|Insufficient|preempt|evict|pressure" | tail -n 10 || true'`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("expected auto-approve for one-line script, got false: %q", cmd)
	}
}

func TestCommandAllowsAutoApprove_KubectlCustomColumnsJSONPathCompound(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	// JSONPath ?(...) must be inside single quotes so bash does not treat it as extglob / glob.
	cmd := `kubectl get nodes -o 'custom-columns=NAME:.metadata.name,CPU_CAP:.status.capacity.cpu,CPU_ALLOC:.status.allocatable.cpu,MEM_CAP:.status.capacity.memory,MEM_ALLOC:.status.allocatable.memory,READY:.status.conditions[?(@.type=="Ready")].status' --no-headers && echo '---' && kubectl top nodes 2>/dev/null | sed -n '1,8p' && echo '---' && kubectl get pods -A --field-selector=status.phase!=Running,status.phase!=Succeeded -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name,PHASE:.status.phase --no-headers | head -n 30`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("want auto-approve for kubectl custom-columns with quoted JSONPath + chain: %q", cmd)
	}
}

func TestCommandAllowsAutoApprove_KubectlTopNodesPodsPlural(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`kubectl top nodes --no-headers 2>/dev/null || echo unavailable`) {
		t.Fatal("kubectl top nodes (plural resource) should match allowlist like top node")
	}
	if !w.CommandAllowsAutoApprove(`kubectl top pods --no-headers 2>/dev/null || true`) {
		t.Fatal("kubectl top pods should match allowlist like top pod")
	}
}
