package hil

import (
	"strings"
	"testing"

	"delve-shell/internal/config"
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
		{[]string{"kubectl", "get", "pods", "--raw=/api/v1/"}, false},
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
