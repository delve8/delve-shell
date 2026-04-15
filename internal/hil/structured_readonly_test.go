package hil

import (
	"strings"
	"testing"

	"delve-shell/internal/config"
	"delve-shell/internal/i18n"
)

func TestStaticSimpleCommandArgs_kubectlCustomColumnsJSONPath(t *testing.T) {
	t.Parallel()
	unquoted := `kubectl get nodes -o custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type==Ready)].status --no-headers`
	if _, ok := staticSimpleCommandArgs(unquoted); ok {
		t.Fatalf("unquoted ?(@...) parses as extglob; expect reject, got ok for %q", unquoted)
	}
	quoted := `kubectl get nodes -o 'custom-columns=NAME:.metadata.name,READY:.status.conditions[?(@.type==Ready)].status' --no-headers`
	args, ok := staticSimpleCommandArgs(quoted)
	if !ok {
		t.Fatalf("staticSimpleCommandArgs failed for %q", quoted)
	}
	if len(args) != 6 || args[0] != "kubectl" || args[1] != "get" || args[2] != "nodes" || args[3] != "-o" {
		t.Fatalf("unexpected argv prefix: %#v", args)
	}
	if !strings.Contains(args[4], "custom-columns=") || !strings.Contains(args[4], "[?(@.type==Ready)]") {
		t.Fatalf("expected JSONPath in arg4, got %q", args[4])
	}
	if args[5] != "--no-headers" {
		t.Fatalf("arg5: %q", args[5])
	}
}

func TestStaticSimpleCommandArgs_quotedOutputWide(t *testing.T) {
	t.Parallel()
	args, ok := staticSimpleCommandArgs(`kubectl get nodes -o 'wide' --no-headers`)
	if !ok || len(args) != 6 || args[4] != "wide" {
		t.Fatalf("got ok=%v args=%#v", ok, args)
	}
}

func TestStaticOrOpaque_kubectlQuotedSimpleNamespace(t *testing.T) {
	t.Parallel()
	pol := config.KubectlReadOnlyCLIPolicyForTest()
	qa, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n "$ns" get pods --no-headers`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if len(qa) < 2 || qa[2].opaque != true {
		t.Fatalf("expected opaque argv slot for -n value, got %#v", qa)
	}
	if !matchReadOnlyCLIArgs(qa, &pol) {
		t.Fatal("expected matchReadOnlyCLIArgs for quoted simple -n value")
	}
}

func TestStaticOrOpaque_rejectsUnquotedOrNonSimpleParam(t *testing.T) {
	t.Parallel()
	if _, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n $ns get pods`); ok {
		t.Fatal("unquoted $ns should be rejected")
	}
	if _, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n "${ns:-x}" get pods`); ok {
		t.Fatal("defaulted param expansion should be rejected")
	}
}

func TestStaticOrOpaque_kubectlQuotedCmdSubstFlagValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["kubectl"]
	if !ok {
		t.Fatal("missing kubectl in default allowlist")
	}
	qa, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n "$(printf '%s' "$ns")" get pods`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if len(qa) < 3 || !qa[2].cmdSubst {
		t.Fatalf("expected quoted cmdSubst argv slot for -n value, got %#v", qa)
	}
	if !matchReadOnlyCLIArgs(qa, &pol) {
		t.Fatal("expected matchReadOnlyCLIArgs for quoted cmdSubst flag value")
	}
}

func TestStaticOrOpaque_kubectlAttachedQuotedCmdSubstFlagValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["kubectl"]
	if !ok {
		t.Fatal("missing kubectl in default allowlist")
	}
	qa, ok := staticOrOpaqueSimpleCommandArgs(`kubectl --namespace="$(printf '%s' "$ns")" get pods`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if len(qa) < 2 || qa[1].flagToken != "--namespace=" || !qa[1].cmdSubst {
		t.Fatalf("expected attached quoted cmdSubst flag token, got %#v", qa)
	}
	if !matchReadOnlyCLIArgs(qa, &pol) {
		t.Fatal("expected attached quoted cmdSubst value to match allowlist")
	}
}

func TestStaticOrOpaque_kubectlShortAttachedQuotedCmdSubstFlagValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["kubectl"]
	if !ok {
		t.Fatal("missing kubectl in default allowlist")
	}
	qa, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n="$(printf '%s' "$ns")" get pods`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if len(qa) < 2 || qa[1].flagToken != "-n=" || !qa[1].cmdSubst {
		t.Fatalf("expected short attached quoted cmdSubst flag token, got %#v", qa)
	}
	if !matchReadOnlyCLIArgs(qa, &pol) {
		t.Fatal("expected short attached quoted cmdSubst value to match allowlist")
	}
}

func TestStaticOrOpaque_crictlQuotedCmdSubstOperandNeedsSentinel(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["crictl"]
	if !ok {
		t.Fatal("missing crictl in default allowlist")
	}
	withoutSentinel, ok := staticOrOpaqueSimpleCommandArgs(`crictl inspect "$(crictl ps -a --name "$p" -q | head -n1)"`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed without sentinel")
	}
	if matchReadOnlyCLIArgs(withoutSentinel, &pol) {
		t.Fatal("quoted cmdSubst operand without -- sentinel should not match")
	}
	withSentinel, ok := staticOrOpaqueSimpleCommandArgs(`crictl inspect -- "$(crictl ps -a --name "$p" -q | head -n1)"`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed with sentinel")
	}
	if !matchReadOnlyCLIArgs(withSentinel, &pol) {
		t.Fatal("quoted cmdSubst operand after -- sentinel should match")
	}
}

func TestStaticOrOpaque_kubectlNestedQuotedCmdSubstFlagValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["kubectl"]
	if !ok {
		t.Fatal("missing kubectl in default allowlist")
	}
	qa, ok := staticOrOpaqueSimpleCommandArgs(`kubectl -n "$(printf '%s' "$(printf default)")" get pods`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if !matchReadOnlyCLIArgs(qa, &pol) {
		t.Fatal("expected nested quoted cmdSubst flag value to match allowlist")
	}
}

func TestStaticOrOpaque_openAnyFlagRejectsQuotedCmdSubstValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["crictl"]
	if !ok {
		t.Fatal("missing crictl in default allowlist")
	}
	separate, ok := staticOrOpaqueSimpleCommandArgs(`crictl ps -a --name "$(printf '%s' "$p")" -q`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed")
	}
	if matchReadOnlyCLIArgs(separate, &pol) {
		t.Fatal("open-any flag value should not accept quoted cmdSubst")
	}
	attached, ok := staticOrOpaqueSimpleCommandArgs(`crictl ps -a --name="$(printf '%s' "$p")" -q`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed for attached value")
	}
	if matchReadOnlyCLIArgs(attached, &pol) {
		t.Fatal("open-any attached flag value should not accept quoted cmdSubst")
	}
}

func TestStaticOrOpaque_openAnyMustNotRejectsQuotedCmdSubstValue(t *testing.T) {
	t.Parallel()
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["sort"]
	if !ok {
		t.Fatal("missing sort in default allowlist")
	}
	separate, ok := staticOrOpaqueSimpleCommandArgs(`sort --output "$(printf '%s' /tmp/out)"`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed for separate value")
	}
	if matchReadOnlyCLIArgs(separate, &pol) {
		t.Fatal("must_not option with separate quoted cmdSubst value should not match")
	}
	attached, ok := staticOrOpaqueSimpleCommandArgs(`sort --output="$(printf '%s' /tmp/out)"`)
	if !ok {
		t.Fatal("staticOrOpaqueSimpleCommandArgs failed for attached value")
	}
	if matchReadOnlyCLIArgs(attached, &pol) {
		t.Fatal("must_not option with attached quoted cmdSubst value should not match")
	}
}

func TestMatchReadOnlyCLIArgv_kubectlPolicy(t *testing.T) {
	pol := config.KubectlReadOnlyCLIPolicyForTest()

	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"kubectl", "get", "pods"}, true},
		{[]string{"kubectl", "-n", "ns", "get", "pods"}, true},
		{[]string{"kubectl", "-n=get", "pods"}, false},
		{[]string{"kubectl", "-n=ns", "get", "pods"}, true},
		{[]string{"kubectl", "--namespace", "x", "--context", "y", "get", "pods"}, true},
		{[]string{"kubectl", "-A", "get", "pods"}, true},
		{[]string{"kubectl", "foo", "get", "pods"}, false},
		{[]string{"kubectl", "--request-timeout=1s", "get", "pods"}, false},
		{[]string{"kubectl", "get", "pods", "--raw=/api/v1/"}, true},
		{[]string{"kubectl", "cluster-info"}, true},
		{[]string{"kubectl", "cluster-info", "dump"}, false},
		{[]string{"kubectl", "cluster-info", "foo", "dump"}, false},
		{[]string{"kubectl", "--help"}, true},
		{[]string{"kubectl", "get", "--help"}, true},
		{[]string{"kubectl", "get", "pods", "-h"}, true},
		{[]string{"/usr/bin/kubectl", "version"}, true},
		{[]string{"kubectl", "version", "--client"}, true},
		{[]string{"kubectl", "version", "-o", "yaml"}, true},
		{[]string{"kubectl", "version", "extra"}, false},
		{[]string{"kubectl", "top", "pod"}, true},
		{[]string{"kubectl", "top", "node"}, true},
		{[]string{"kubectl", "top", "pods"}, true},
		{[]string{"kubectl", "top", "nodes"}, true},
		{[]string{"kubectl", "top", "pod", "-A", "--no-headers"}, true},
		{[]string{"kubectl", "top", "pods", "-A", "--no-headers"}, true},
		{[]string{"kubectl", "config", "view", "--raw"}, true},
		{[]string{"helm", "version"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_rootMustBeforeSubcommand(t *testing.T) {
	pol := config.ReadOnlyCLIPolicy{
		Name: "tool",
		Root: &config.RootSpec{
			Flags: config.NewFlagAllow([]config.AllowedOption{
				{Short: "v", Value: "none"},
			}).WithMust([]config.AllowedOption{{Short: "v"}}),
			Operands: config.NewOperandsNone(),
			Subcommands: config.SubcommandMap{
				"get": {},
			},
		},
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"tool", "-v", "get"}, true},
		{[]string{"tool", "get"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_mustNotListedInAllowStillConsumable(t *testing.T) {
	pol := config.ReadOnlyCLIPolicy{
		Name: "tool",
		Root: &config.RootSpec{
			Flags: config.NewFlagAllow([]config.AllowedOption{
				{Short: "b", Value: "none"},
				{Short: "c", Value: "none"},
			}).WithMust([]config.AllowedOption{{Short: "a", Value: "none"}}),
			Operands: config.NewOperandsAny(),
		},
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"tool", "-a", "op"}, true},
		{[]string{"tool", "-a", "-b", "op"}, true},
		{[]string{"tool", "-b", "op"}, false},
		{[]string{"tool", "op"}, false},
		{[]string{"tool", "-x"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_jqMustNotFromFile(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["jq"]
	if !ok {
		t.Fatal("missing jq in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"jq", "-r", ".items[]"}, true},
		{[]string{"jq", "-f", "prog.jq", "."}, false},
		{[]string{"jq", "--from-file", "prog.jq", "."}, false},
		{[]string{"jq", "--from-file=prog.jq", "."}, false},
		{[]string{"jq", "-rf", "."}, false},
		{[]string{"jq", "-r", "-f", "x"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
	if !pol.PermissiveVarArgs() {
		t.Fatal("jq policy should be permissive for var-args auto-approve")
	}
}

func TestMatchReadOnlyCLIArgv_sedSortMustNot(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	sedPol := ld.Commands["sed"]
	sortPol := ld.Commands["sort"]
	cases := []struct {
		name string
		pol  config.ReadOnlyCLIPolicy
		args []string
		want bool
	}{
		{"sed ok", sedPol, []string{"sed", "-n", "1p"}, true},
		{"sed -i", sedPol, []string{"sed", "-i", "s/a/b/", "f"}, false},
		{"sed -i.bak", sedPol, []string{"sed", "-i.bak", "s/a/b/", "f"}, false},
		{"sed --in-place", sedPol, []string{"sed", "--in-place", "-n", "1p", "x"}, false},
		{"sort ok", sortPol, []string{"sort", "-nr"}, true},
		{"sort -o", sortPol, []string{"sort", "-o", "/tmp/out"}, false},
		{"sort --output", sortPol, []string{"sort", "--output", "/tmp/out"}, false},
	}
	for _, tt := range cases {
		got := MatchReadOnlyCLIArgv(tt.args, &tt.pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_commandMustDashV(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["command"]
	if !ok {
		t.Fatal("missing command in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"command", "-v", "kubectl"}, true},
		{[]string{"command", "-v", "sh", "bash"}, true},
		{[]string{"command", "kubectl"}, false},
		{[]string{"command", "-V", "kubectl"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_findSingleDashPredicates(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["find"]
	if !ok {
		t.Fatal("missing find in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"find", ".", "-name", "*.log", "-type", "f"}, true},
		{[]string{"find", "/var/log", "-maxdepth", "2", "-iname", "*api*"}, true},
		{[]string{"find", ".", "--name", "*.log"}, false},
		{[]string{"find", ".", "-exec", "rm", "{}", ";"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestXargsReadOnlySegmentOK(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	tests := []struct {
		name string
		seg  string
		want bool
	}{
		{"kubectl get pod with sentinel", `xargs -r -n1 kubectl get pod --`, true},
		{"systemctl status with sentinel", `xargs --no-run-if-empty --max-args=2 systemctl status --`, true},
		{"missing sentinel", `xargs -r -n1 kubectl get pod`, false},
		{"replacement template", `xargs -I{} kubectl get pod {}`, false},
		{"shell wrapper target", `xargs -r sh -c 'echo "$1"' --`, false},
		{"target utility not allowlisted", `xargs -r -n1 rm --`, false},
		{"dynamic subcommand slot not fixed", `xargs -r -n1 kubectl --`, false},
		{"dangerous flag -P", `xargs -P 4 kubectl get pod --`, false},
		{"dangerous flag -L", `xargs -L 2 kubectl get pod --`, false},
		{"dangerous flag -a", `xargs -a names.txt kubectl get pod --`, false},
		{"dangerous flag -d", `xargs -d , kubectl get pod --`, false},
		{"unknown xargs flag", `xargs -P 4 kubectl get pod --`, false},
	}
	for _, tt := range tests {
		args, ok := staticSimpleCommandArgs(tt.seg)
		if !ok {
			if tt.want {
				t.Fatalf("%s: staticSimpleCommandArgs failed for %q", tt.name, tt.seg)
			}
			continue
		}
		got := xargsReadOnlySegmentOK(args, ld.Commands)
		if got != tt.want {
			t.Fatalf("%s: xargsReadOnlySegmentOK(%q) = %v, want %v", tt.name, tt.seg, got, tt.want)
		}
	}
}

func TestXargsReadOnlySegmentReason(t *testing.T) {
	i18n.SetLang("en")
	ld := config.DefaultLoadedAllowlist()
	tests := []struct {
		name string
		seg  string
		want string
	}{
		{"unsafe flag", `xargs -P 4 kubectl get pod --`, i18n.T(i18n.KeyAutoApproveHLXargsUnsafeFlag)},
		{"missing target", `xargs -r -n1`, i18n.T(i18n.KeyAutoApproveHLXargsMissingTarget)},
		{"missing sentinel", `xargs -r -n1 kubectl get pod`, i18n.T(i18n.KeyAutoApproveHLXargsMissingSentinel)},
		{"unsafe target", `xargs -r sh -c 'echo "$1"' --`, i18n.T(i18n.KeyAutoApproveHLXargsUnsafeTarget)},
		{"target mismatch", `xargs -r -n1 kubectl --`, i18n.T(i18n.KeyAutoApproveHLXargsTargetMismatch)},
		{"ok", `xargs -r -n1 kubectl get pod --`, ""},
	}
	for _, tt := range tests {
		args, ok := staticSimpleCommandArgs(tt.seg)
		if !ok {
			t.Fatalf("%s: staticSimpleCommandArgs failed for %q", tt.name, tt.seg)
		}
		got := xargsReadOnlySegmentReason(args, ld.Commands)
		if got != tt.want {
			t.Fatalf("%s: xargsReadOnlySegmentReason(%q) = %q, want %q", tt.name, tt.seg, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_envNoExec(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["env"]
	if !ok {
		t.Fatal("missing env in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"env"}, true},
		{[]string{"env", "PATH=/tmp"}, false},
		{[]string{"env", "sh", "-c", "echo hi"}, false},
		{[]string{"env", "-i"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_gitReadOnlyTightened(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["git"]
	if !ok {
		t.Fatal("missing git in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"git", "branch"}, true},
		{[]string{"git", "branch", "--all"}, true},
		{[]string{"git", "branch", "--show-current"}, true},
		{[]string{"git", "branch", "feature/test"}, false},
		{[]string{"git", "remote"}, true},
		{[]string{"git", "remote", "-v"}, true},
		{[]string{"git", "remote", "show", "origin"}, true},
		{[]string{"git", "remote", "add", "origin", "git@github.com:o/r.git"}, false},
		{[]string{"git", "tag"}, true},
		{[]string{"git", "tag", "--list"}, true},
		{[]string{"git", "tag", "v1.2.3"}, false},
		{[]string{"git", "reflog"}, true},
		{[]string{"git", "reflog", "show", "HEAD"}, true},
		{[]string{"git", "reflog", "expire", "--all"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_dmesgSysctlMustNot(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	dmesgPol := ld.Commands["dmesg"]
	sysctlPol := ld.Commands["sysctl"]
	cases := []struct {
		name string
		pol  config.ReadOnlyCLIPolicy
		args []string
		want bool
	}{
		{"dmesg ok", dmesgPol, []string{"dmesg", "-T"}, true},
		{"dmesg clear", dmesgPol, []string{"dmesg", "-C"}, false},
		{"dmesg read-clear", dmesgPol, []string{"dmesg", "--read-clear"}, false},
		{"sysctl read", sysctlPol, []string{"sysctl", "kernel.hostname"}, true},
		{"sysctl all", sysctlPol, []string{"sysctl", "-a"}, true},
		{"sysctl write", sysctlPol, []string{"sysctl", "-w", "net.ipv4.ip_forward=1"}, false},
		{"sysctl load", sysctlPol, []string{"sysctl", "-p", "/etc/sysctl.conf"}, false},
		{"sysctl system", sysctlPol, []string{"sysctl", "--system"}, false},
	}
	for _, tt := range cases {
		got := MatchReadOnlyCLIArgv(tt.args, &tt.pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_ipHelmAndNerdctlReadOnly(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	ipPol := ld.Commands["ip"]
	helmPol := ld.Commands["helm"]
	nerdctlPol := ld.Commands["nerdctl"]
	cases := []struct {
		name string
		pol  config.ReadOnlyCLIPolicy
		args []string
		want bool
	}{
		{"ip neigh show", ipPol, []string{"ip", "neigh", "show"}, true},
		{"ip rule list", ipPol, []string{"ip", "rule", "list"}, true},
		{"ip rule add", ipPol, []string{"ip", "rule", "add", "from", "10.0.0.0/8"}, false},
		{"helm version", helmPol, []string{"helm", "version"}, true},
		{"helm list", helmPol, []string{"helm", "list", "-A"}, true},
		{"helm ls alias", helmPol, []string{"helm", "ls", "-A"}, true},
		{"helm get values", helmPol, []string{"helm", "get", "values", "release-a"}, true},
		{"helm install", helmPol, []string{"helm", "install", "release-a", "./chart"}, false},
		{"nerdctl ps", nerdctlPol, []string{"nerdctl", "ps"}, true},
		{"nerdctl namespace ps", nerdctlPol, []string{"nerdctl", "--namespace", "k8s.io", "ps"}, true},
		{"nerdctl container logs", nerdctlPol, []string{"nerdctl", "container", "logs", "abc123"}, true},
		{"nerdctl container logs follow", nerdctlPol, []string{"nerdctl", "container", "logs", "--follow", "abc123"}, true},
		{"nerdctl run", nerdctlPol, []string{"nerdctl", "run", "nginx"}, false},
	}
	for _, tt := range cases {
		got := MatchReadOnlyCLIArgv(tt.args, &tt.pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_topMustBeBoundedBatch(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	pol, ok := ld.Commands["top"]
	if !ok {
		t.Fatal("missing top in default allowlist")
	}
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"top", "-b", "-n", "1"}, true},
		{[]string{"top", "-b", "-n", "1", "-o", "%CPU"}, true},
		{[]string{"top"}, false},
		{[]string{"top", "-b"}, false},
		{[]string{"top", "-n", "1"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_secondBatchReadOnlyCommands(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	cases := []struct {
		name   string
		policy string
		args   []string
		wantOK bool
	}{
		{"apt list", "apt", []string{"apt", "list", "--installed"}, true},
		{"apt install blocked", "apt", []string{"apt", "install", "nginx"}, false},
		{"apt-cache policy", "apt-cache", []string{"apt-cache", "policy", "nginx"}, true},
		{"dpkg list", "dpkg", []string{"dpkg", "-l", "bash"}, true},
		{"dpkg configure blocked", "dpkg", []string{"dpkg", "--configure", "-a"}, false},
		{"docker logs", "docker", []string{"docker", "logs", "--tail", "100", "c1"}, true},
		{"docker logs follow", "docker", []string{"docker", "logs", "--follow", "c1"}, true},
		{"docker container logs", "docker", []string{"docker", "container", "logs", "--tail", "100", "c1"}, true},
		{"docker container logs follow", "docker", []string{"docker", "container", "logs", "--follow", "c1"}, true},
		{"docker stats no-stream", "docker", []string{"docker", "stats", "--no-stream", "c1"}, true},
		{"docker stats blocked", "docker", []string{"docker", "stats", "c1"}, false},
		{"kubectl events", "kubectl", []string{"kubectl", "events", "--for", "pod/x"}, true},
		{"kubectl get raw", "kubectl", []string{"kubectl", "get", "--raw", "/readyz"}, true},
		{"loginctl list-sessions", "loginctl", []string{"loginctl", "list-sessions"}, true},
		{"loginctl terminate blocked", "loginctl", []string{"loginctl", "terminate-session", "2"}, false},
		{"timedatectl status", "timedatectl", []string{"timedatectl", "status"}, true},
		{"timedatectl set-time blocked", "timedatectl", []string{"timedatectl", "set-time", "2026-01-01"}, false},
		{"hostnamectl status", "hostnamectl", []string{"hostnamectl", "status"}, true},
		{"hostnamectl set-hostname blocked", "hostnamectl", []string{"hostnamectl", "set-hostname", "x"}, false},
		{"resolvectl status", "resolvectl", []string{"resolvectl", "status"}, true},
		{"resolvectl flush blocked", "resolvectl", []string{"resolvectl", "flush-caches"}, false},
		{"systemd-resolve status", "systemd-resolve", []string{"systemd-resolve", "--status"}, true},
		{"systemd-resolve flush blocked", "systemd-resolve", []string{"systemd-resolve", "--flush-caches"}, false},
		{"brew version", "brew", []string{"brew", "--version"}, true},
		{"brew install blocked", "brew", []string{"brew", "install", "jq"}, false},
	}
	for _, tt := range cases {
		pol, ok := ld.Commands[tt.policy]
		if !ok {
			t.Fatalf("%s: missing policy %q", tt.name, tt.policy)
		}
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.wantOK {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.wantOK)
		}
	}
}

func TestMatchReadOnlyCLIArgv_thirdBatchInfraReadOnlyCommands(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	cases := []struct {
		name   string
		policy string
		args   []string
		want   bool
	}{
		{"blkid", "blkid", []string{"blkid"}, true},
		{"bridge fdb show", "bridge", []string{"bridge", "fdb", "show"}, true},
		{"bridge vlan add blocked", "bridge", []string{"bridge", "vlan", "add", "dev", "eth0", "vid", "100"}, false},
		{"dmidecode type", "dmidecode", []string{"dmidecode", "--type", "memory"}, true},
		{"dmidecode dump-bin blocked", "dmidecode", []string{"dmidecode", "--dump-bin", "/tmp/dmi.bin"}, false},
		{"dnf list", "dnf", []string{"dnf", "list", "installed"}, true},
		{"dnf install blocked", "dnf", []string{"dnf", "install", "vim"}, false},
		{"ethtool i", "ethtool", []string{"ethtool", "-i", "eth0"}, true},
		{"ethtool set blocked", "ethtool", []string{"ethtool", "-K", "eth0", "gro", "off"}, false},
		{"iptables list", "iptables", []string{"iptables", "-L", "-n", "-v"}, true},
		{"iptables save", "iptables-save", []string{"iptables-save"}, true},
		{"iptables append blocked", "iptables", []string{"iptables", "-A", "INPUT", "-j", "ACCEPT"}, false},
		{"ip6tables save", "ip6tables-save", []string{"ip6tables-save"}, true},
		{"nft list ruleset", "nft", []string{"nft", "list", "ruleset"}, true},
		{"nft add blocked", "nft", []string{"nft", "add", "table", "inet", "x"}, false},
		{"losetup list", "losetup", []string{"losetup", "-l"}, true},
		{"losetup detach blocked", "losetup", []string{"losetup", "-d", "/dev/loop0"}, false},
		{"numactl hardware", "numactl", []string{"numactl", "--hardware"}, true},
		{"numactl cpubind blocked", "numactl", []string{"numactl", "--cpubind=0", "cmd"}, false},
		{"numastat", "numastat", []string{"numastat"}, true},
		{"nmcli general status", "nmcli", []string{"nmcli", "general", "status"}, true},
		{"nmcli connection show", "nmcli", []string{"nmcli", "connection", "show"}, true},
		{"nmcli connection profile detail", "nmcli", []string{"nmcli", "connection", "show", "prod-wifi"}, true},
		{"nmcli up blocked", "nmcli", []string{"nmcli", "connection", "up", "eth0"}, false},
		{"podman container logs", "podman", []string{"podman", "container", "logs", "--tail", "100", "c1"}, true},
		{"podman container logs follow", "podman", []string{"podman", "container", "logs", "--follow", "c1"}, true},
		{"rpm query all", "rpm", []string{"rpm", "-q", "-a"}, true},
		{"rpm install blocked", "rpm", []string{"rpm", "-i", "pkg.rpm"}, false},
		{"swapon summary", "swapon", []string{"swapon", "-s"}, true},
		{"swapon enable blocked", "swapon", []string{"swapon", "/swapfile"}, false},
		{"tc qdisc show", "tc", []string{"tc", "qdisc", "show", "dev", "eth0"}, true},
		{"tc add blocked", "tc", []string{"tc", "qdisc", "add", "dev", "eth0", "root", "fq_codel"}, false},
		{"virsh list", "virsh", []string{"virsh", "list", "--all"}, true},
		{"virsh start blocked", "virsh", []string{"virsh", "start", "vm1"}, false},
		{"yum search", "yum", []string{"yum", "search", "bash"}, true},
		{"yum install blocked", "yum", []string{"yum", "install", "bash"}, false},
	}
	for _, tt := range cases {
		pol, ok := ld.Commands[tt.policy]
		if !ok {
			t.Fatalf("%s: missing policy %q", tt.name, tt.policy)
		}
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_fourthBatchCloudReadOnlyCommands(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	cases := []struct {
		name   string
		policy string
		args   []string
		want   bool
	}{
		{"aws caller identity", "aws", []string{"aws", "sts", "get-caller-identity"}, true},
		{"aws profile ec2 describe", "aws", []string{"aws", "--profile", "prod", "ec2", "describe-instances"}, true},
		{"aws s3 ls", "aws", []string{"aws", "s3", "ls", "s3://bucket-a"}, true},
		{"aws s3 cp blocked", "aws", []string{"aws", "s3", "cp", "a", "b"}, false},
		{"aws ecr auth token", "aws", []string{"aws", "ecr", "get-authorization-token"}, true},
		{"aws eks describe", "aws", []string{"aws", "eks", "describe-cluster", "--name", "prod"}, true},
		{"aws eks update-kubeconfig blocked", "aws", []string{"aws", "eks", "update-kubeconfig", "--name", "prod"}, false},
		{"aws ec2 terminate blocked", "aws", []string{"aws", "ec2", "terminate-instances", "--instance-ids", "i-1"}, false},
		{"gcloud auth list", "gcloud", []string{"gcloud", "auth", "list"}, true},
		{"gcloud project instances list", "gcloud", []string{"gcloud", "--project", "p1", "compute", "instances", "list"}, true},
		{"gcloud gke describe", "gcloud", []string{"gcloud", "container", "clusters", "describe", "c1", "--region", "us-central1"}, true},
		{"gcloud get-credentials blocked", "gcloud", []string{"gcloud", "container", "clusters", "get-credentials", "c1"}, false},
		{"gcloud delete blocked", "gcloud", []string{"gcloud", "compute", "instances", "delete", "vm1"}, false},
		{"az account show", "az", []string{"az", "account", "show"}, true},
		{"az resource list", "az", []string{"az", "resource", "list", "-g", "rg1"}, true},
		{"az aks show", "az", []string{"az", "aks", "show", "-g", "rg1", "-n", "aks1"}, true},
		{"az aks get-credentials blocked", "az", []string{"az", "aks", "get-credentials", "-g", "rg1", "-n", "aks1"}, false},
		{"az group delete blocked", "az", []string{"az", "group", "delete", "-n", "rg1"}, false},
	}
	for _, tt := range cases {
		pol, ok := ld.Commands[tt.policy]
		if !ok {
			t.Fatalf("%s: missing policy %q", tt.name, tt.policy)
		}
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestMatchReadOnlyCLIArgv_namespaceParentsDoNotPermitUnknownWriteSubcommands(t *testing.T) {
	ld := config.DefaultLoadedAllowlist()
	cases := []struct {
		name   string
		policy string
		args   []string
		want   bool
	}{
		{"docker compose up blocked", "docker", []string{"docker", "compose", "up", "-d"}, false},
		{"docker network rm blocked", "docker", []string{"docker", "network", "rm", "net1"}, false},
		{"docker volume create blocked", "docker", []string{"docker", "volume", "create", "v1"}, false},
		{"docker system prune blocked", "docker", []string{"docker", "system", "prune", "-f"}, false},
		{"podman compose up blocked", "podman", []string{"podman", "compose", "up", "-d"}, false},
		{"podman network rm blocked", "podman", []string{"podman", "network", "rm", "net1"}, false},
		{"podman volume create blocked", "podman", []string{"podman", "volume", "create", "v1"}, false},
		{"nerdctl image pull blocked", "nerdctl", []string{"nerdctl", "image", "pull", "nginx:latest"}, false},
		{"nerdctl network create blocked", "nerdctl", []string{"nerdctl", "network", "create", "net1"}, false},
		{"ctr images pull blocked", "ctr", []string{"ctr", "images", "pull", "docker.io/library/nginx:latest"}, false},
		{"ctr tasks exec blocked", "ctr", []string{"ctr", "tasks", "exec", "--exec-id", "x", "c1", "sh"}, false},
		{"ctr snapshot rm blocked", "ctr", []string{"ctr", "snapshot", "rm", "snap1"}, false},
		{"crictl image pull blocked", "crictl", []string{"crictl", "image", "pull", "nginx:latest"}, false},
		{"crictl img pull blocked", "crictl", []string{"crictl", "img", "pull", "nginx:latest"}, false},
	}
	for _, tt := range cases {
		pol, ok := ld.Commands[tt.policy]
		if !ok {
			t.Fatalf("%s: missing policy %q", tt.name, tt.policy)
		}
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("%s: MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.name, tt.args, got, tt.want)
		}
	}
}
