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

func TestCommandAllowsAutoApprove_StaticAssignmentResolvesQuotedSimpleParam(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`verb=get; kubectl "$verb" pods`) {
		t.Fatal("quoted simple param in subcommand position should resolve from prior static assignment")
	}
	if w.CommandAllowsAutoApprove(`verb=delete; kubectl "$verb" pods`) {
		t.Fatal("resolved static assignment must still reject disallowed values")
	}
}

func TestCommandAllowsAutoApprove_ForLoopLiteralValuesResolveQuotedSimpleParam(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`for verb in get describe; do kubectl "$verb" pods; done`) {
		t.Fatal("for-in literal values should resolve quoted simple param in subcommand position")
	}
	if w.CommandAllowsAutoApprove(`for verb in get delete; do kubectl "$verb" pods; done`) {
		t.Fatal("every inferred for-in value must satisfy allowlist")
	}
	if w.CommandAllowsAutoApprove(`for verb in get delete; do :; done; kubectl "$verb" pods`) {
		t.Fatal("post-loop inference should preserve shell's last iteration value")
	}
}

func TestCommandAllowsAutoApprove_ReadInvalidatesStaticAssignment(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`verb=get; read verb; kubectl "$verb" pods`) {
		t.Fatal("read should invalidate prior static assignment for the same variable")
	}
}

func TestCommandAllowsAutoApprove_StaticAliasResolvesQuotedSimpleParam(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`verb=get; alias="$verb"; kubectl "$alias" pods`) {
		t.Fatal("quoted simple param assignment should propagate known static values")
	}
	if w.CommandAllowsAutoApprove(`verb=delete; alias="$verb"; kubectl "$alias" pods`) {
		t.Fatal("propagated alias values must still satisfy allowlist")
	}
}

func TestCommandAllowsAutoApprove_DynamicOrUnsupportedAssignmentInvalidatesStaticValue(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	for _, cmd := range []string{
		`verb=get; verb="$unknown"; kubectl "$verb" pods`,
		`verb=get; verb=$(printf get); kubectl "$verb" pods`,
		`verb=get; verb+=x; kubectl "$verb" pods`,
		`verb=get; verb[0]=get; kubectl "$verb" pods`,
	} {
		if w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("unsupported or dynamic reassignment must invalidate prior static value: %q", cmd)
		}
	}
}

func TestCommandAllowsAutoApprove_IfEqualityNarrowsQuotedSimpleParam(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`if [ "$verb" = get ]; then kubectl "$verb" pods; fi`) {
		t.Fatal("POSIX [ equality should narrow quoted simple param inside then branch")
	}
	if !w.CommandAllowsAutoApprove(`if test get = "$verb"; then kubectl "$verb" pods; fi`) {
		t.Fatal("test equality should narrow quoted simple param when literal is on the left")
	}
	if !w.CommandAllowsAutoApprove(`if [[ "$verb" == describe ]]; then kubectl "$verb" pods; fi`) {
		t.Fatal("[[ equality should narrow quoted simple param inside then branch")
	}
	if w.CommandAllowsAutoApprove(`if [ "$verb" = delete ]; then kubectl "$verb" pods; fi`) {
		t.Fatal("narrowed if equality value must still satisfy allowlist")
	}
	if w.CommandAllowsAutoApprove(`if [[ "$verb" == get* ]]; then kubectl "$verb" pods; fi`) {
		t.Fatal("[[ pattern match must not be treated as an exact static value")
	}
}

func TestCommandAllowsAutoApprove_IfUnsupportedConditionDoesNotNarrow(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	for _, cmd := range []string{
		`if [ "$verb" != get ]; then kubectl "$verb" pods; fi`,
		`if [[ "$verb" != get ]]; then kubectl "$verb" pods; fi`,
		`if [ "$verb" = get ] && [ "$ns" = default ]; then kubectl "$verb" pods; fi`,
		`if [ "$verb" = "$other" ]; then kubectl "$verb" pods; fi`,
	} {
		if w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("unsupported condition form must not narrow and auto-approve: %q", cmd)
		}
	}
}

func TestCommandAllowsAutoApprove_CaseLiteralPatternNarrowsQuotedSimpleParam(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`case "$verb" in get|describe) kubectl "$verb" pods ;; esac`) {
		t.Fatal("literal case patterns should narrow quoted simple param inside item body")
	}
	if w.CommandAllowsAutoApprove(`case "$verb" in get|delete) kubectl "$verb" pods ;; esac`) {
		t.Fatal("every inferred case pattern value must satisfy allowlist")
	}
	if w.CommandAllowsAutoApprove(`case "$verb" in get*) kubectl "$verb" pods ;; esac`) {
		t.Fatal("case glob patterns must not be treated as exact static values")
	}
}

func TestCommandAllowsAutoApprove_CaseMixedAndNonTargetFormsStayConservative(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`case "$verb" in get) kubectl "$verb" pods ;; delete) kubectl "$verb" pods ;; esac`) {
		t.Fatal("mixed case items must reject when any inferred branch value is unsafe")
	}
	if w.CommandAllowsAutoApprove(`verb=delete; case "$verb" in "$verb") kubectl "$verb" pods ;; esac`) {
		t.Fatal("dynamic case patterns must not bypass allowlist when they resolve to unsafe values")
	}
	if w.CommandAllowsAutoApprove(`case get in get) kubectl "$verb" pods ;; esac`) {
		t.Fatal("case narrowing should not apply when case word is not a simple quoted parameter")
	}
}

func TestCommandAllowsAutoApprove_UnsafeInferredValuesStillRejectInsideCommandSubstitution(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`verb=delete; echo "$(kubectl "$verb" pods)"`) {
		t.Fatal("unsafe inferred values inside command substitutions must still reject")
	}
}

func TestCommandAllowsAutoApprove_VariantExpansionLimitStaysConservative(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := `for verb in get describe logs top api-resources api-versions cluster-info explain version auth attach cp exec diff patch apply delete cordon uncordon drain taint label annotate rollout scale autoscale set wait debug kustomize plugin config completion options; do kubectl "$verb" pods; done`
	if w.CommandAllowsAutoApprove(cmd) {
		t.Fatal("too many inferred variants must fail closed")
	}
}

func TestCommandAllowsAutoApprove_AwkDoubleQuotedFieldRefNotRejectedAsShellVar(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	// Double-quoted awk program: $1 is shell ParamExp; awk must be permissiveArgv0 so collectAllowlistSegments does not reject.
	if !w.CommandAllowsAutoApprove(`awk "{print $1}" </dev/null`) {
		t.Fatal("want auto-approve: $1 is awk field ref inside double quotes, not a kubectl-style expansion risk")
	}
}

func TestCommandAllowsAutoApprove_GawkMawkNawkBenignLikeAwk(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	for _, cmd := range []string{
		`gawk '{print NR}'`,
		`mawk '{print NR}'`,
		`nawk '{print NR}'`,
	} {
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("want auto-approve for read-only awk-family program: %q", cmd)
		}
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

func TestCommandAllowsAutoApprove_commandDashVOnly(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`command -v kubectl`) {
		t.Fatal("command -v should auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`command -v sh bash`) {
		t.Fatal("command -v with multiple names should auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`command -v "$PATH"`) {
		t.Fatal("command -v with quoted simple param should auto-approve")
	}
	if w.CommandAllowsAutoApprove(`command kubectl`) {
		t.Fatal("command without -v runs utility; must not auto-approve")
	}
	if w.CommandAllowsAutoApprove(`command -V kubectl`) {
		t.Fatal("command -V not allowed; must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_exitAllowlistPermissive(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	for _, cmd := range []string{
		`exit`,
		`exit 0`,
		`exit 255`,
		`exit "$?"`,
		`exit $(echo 1)`,
		`exit kubectl`,
		`exit 0 1`,
	} {
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("expected auto-approve for permissive allowlisted exit: %q", cmd)
		}
	}
}

func TestCommandAllowsAutoApprove_whileReadMultiVarThenKubectlOpaqueNS(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := `kubectl top nodes --no-headers 2>/dev/null | awk '{print $1, $3, $5}' | while read n cpu mem; do kubectl -n "$n" get pods --no-headers; done`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("expected auto-approve: read adds no segment; kubectl -n uses quoted simple param\n%q", cmd)
	}
}

func TestCommandAllowsAutoApprove_ForLoopKubectlLookupWithAwkVar(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := `for p in metis archon-manager updater-manager sentry capi-controller-manager kube-scheduler kube-controller-manager; do ns=$(kubectl get pod -A --no-headers 2>/dev/null | awk -v n="$p" '$2==n{print $1; exit}'); [ -z "$ns" ] && continue; echo "### $ns/$p"; kubectl get pod -n "$ns" "$p" -o jsonpath='{range .status.containerStatuses[*]}{.name}{" restart="}{.restartCount}{" lastReason="}{.lastState.terminated.reason}{" exit="}{.lastState.terminated.exitCode}{" finishedAt="}{.lastState.terminated.finishedAt}{"\n"}{end}'; echo; done`
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatalf("expected auto-approve for read-only kubectl/awk for-loop lookup:\n%q\nspans=%#v", cmd, w.CommandAutoApproveHighlight(cmd))
	}
}

func TestAwkBenignRejectReason_VarAndExit(t *testing.T) {
	seg := `awk -v n="$p" '$2==n{print $1; exit}'`
	isAwk, reason := awkBenignRejectReason(seg)
	if !isAwk || reason != "" {
		t.Fatalf("expected benign awk, got isAwk=%v reason=%q", isAwk, reason)
	}
}

func TestCommandAllowsAutoApprove_readRejectsDisallowedExpansionInArgs(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`read -p "$(echo x)" a`) {
		t.Fatal("read with command substitution in args must not auto-approve without checking inner cmd")
	}
}

func TestCommandAllowsAutoApprove_KubectlQuotedSimpleVarOpaque(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`bash -lc 'kubectl get ns -n "${NS}"'`) {
		t.Fatal("kubectl -n with double-quoted simple parameter should auto-approve (flag value slot unconstrained)")
	}
	if w.CommandAllowsAutoApprove(`bash -lc 'kubectl get ns -n ${NS}'`) {
		t.Fatal("unquoted expansion in kubectl args must not auto-approve")
	}
	if w.CommandAllowsAutoApprove(`bash -lc 'kubectl get ns -n "${NS:-x}"'`) {
		t.Fatal("non-simple parameter expansion in quotes must not auto-approve")
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

func TestCommandAllowsAutoApprove_KubectlQuotedCmdSubstFlagValue(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`kubectl -n "$(printf '%s' "$ns")" get pods`) {
		t.Fatal("quoted command substitution in concrete flag value slot should auto-approve when inner command is read-only")
	}
	if !w.CommandAllowsAutoApprove(`kubectl -n="$(printf '%s' "$ns")" get pods`) {
		t.Fatal("short attached quoted command substitution flag value should auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`kubectl --namespace="$(printf '%s' "$ns")" get pods`) {
		t.Fatal("attached quoted command substitution flag value should auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`kubectl -n "$(printf '%s' "$(printf default)")" get pods`) {
		t.Fatal("nested quoted command substitution in concrete flag value slot should auto-approve")
	}
	if w.CommandAllowsAutoApprove(`kubectl -n $(printf '%s' "$ns") get pods`) {
		t.Fatal("unquoted command substitution in flag value must not auto-approve")
	}
	if w.CommandAllowsAutoApprove(`kubectl -n "$(rm -rf /)" get pods`) {
		t.Fatal("quoted command substitution must not auto-approve when inner command is unsafe")
	}
	if w.CommandAllowsAutoApprove(`kubectl "$(printf get)" pods`) {
		t.Fatal("command substitution in subcommand position must not auto-approve")
	}
	if w.CommandAllowsAutoApprove(`kubectl "$(printf -- --namespace=default)" get pods`) {
		t.Fatal("command substitution that could synthesize a flag name must not auto-approve")
	}
}

func TestCommandAllowsAutoApprove_AssignmentCmdSubstInnerOnly(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`cid=$(crictl ps -a --name "$p" -q | head -n1)`) {
		t.Fatal("assignment command substitution should auto-approve when inner commands are read-only")
	}
	if !w.CommandAllowsAutoApprove(`if cid=$(crictl ps -a --name "$p" -q | head -n1); then echo "$cid"; fi`) {
		t.Fatal("control-flow should still recurse into safe assignment command substitution")
	}
	if w.CommandAllowsAutoApprove(`cid=$(rm -rf /)`) {
		t.Fatal("assignment command substitution must not auto-approve when inner command is unsafe")
	}
	if w.CommandAllowsAutoApprove(`if cid=$(rm -rf /); then echo "$cid"; fi`) {
		t.Fatal("control-flow should still reject unsafe assignment command substitution")
	}
}

func TestCommandAllowsAutoApprove_OpenAnyFlagRejectsQuotedCmdSubstValue(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`crictl ps -a --name "$(printf '%s' "$p")" -q`) {
		t.Fatal("open-any flag policies must not auto-approve quoted command substitution values")
	}
	if w.CommandAllowsAutoApprove(`crictl ps -a --name="$(printf '%s' "$p")" -q`) {
		t.Fatal("open-any flag policies must not auto-approve attached quoted command substitution values")
	}
}

func TestCommandAllowsAutoApprove_CrictlQuotedCmdSubstOperandNeedsSentinel(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if w.CommandAllowsAutoApprove(`crictl inspect "$(crictl ps -a --name "$p" -q | head -n1)"`) {
		t.Fatal("quoted command substitution operand without -- sentinel must not auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`crictl inspect -- "$(crictl ps -a --name "$p" -q | head -n1)"`) {
		t.Fatal("quoted command substitution operand after -- sentinel should auto-approve")
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
	if !w.CommandAllowsAutoApprove(`kubectl top pod -A --no-headers 2>/dev/null`) {
		t.Fatal("kubectl top pod with -A after resource should auto-approve (global flag merged at leaf)")
	}
}

func TestCommandAllowsAutoApprove_KubectlVersionShort(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`kubectl version --client --short 2>/dev/null`) {
		t.Fatal("kubectl version --client --short should auto-approve")
	}
}

func TestCommandAllowsAutoApprove_XargsRestrictedSubset(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	if !w.CommandAllowsAutoApprove(`printf '%s\n' pod-a pod-b | xargs -r -n1 kubectl get pod --`) {
		t.Fatal("restricted xargs with fixed kubectl prefix and -- sentinel should auto-approve")
	}
	if !w.CommandAllowsAutoApprove(`printf '%s\000' sshd kubelet | xargs -0 --no-run-if-empty --max-args=2 systemctl status --`) {
		t.Fatal("restricted xargs with systemctl status and -- sentinel should auto-approve")
	}
	if w.CommandAllowsAutoApprove(`printf '%s\n' delete pod-a | xargs -r -n1 kubectl --`) {
		t.Fatal("xargs must not auto-approve when stdin can fill kubectl subcommand slot")
	}
	if w.CommandAllowsAutoApprove(`printf '%s\n' pod-a | xargs -r -n1 kubectl get pod`) {
		t.Fatal("xargs without target -- sentinel must require approval")
	}
	if w.CommandAllowsAutoApprove(`printf '%s\n' pod-a | xargs -I{} kubectl get pod {}`) {
		t.Fatal("xargs replacement mode must require approval")
	}
	if w.CommandAllowsAutoApprove(`printf '%s\n' hi | xargs -r sh -c 'echo "$1"' --`) {
		t.Fatal("xargs shell-wrapper target must require approval")
	}
}
