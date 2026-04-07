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
		{[]string{"helm", "version"}, false},
	}
	for _, tt := range tests {
		got := MatchReadOnlyCLIArgv(tt.args, &pol)
		if got != tt.want {
			t.Errorf("MatchReadOnlyCLIArgv(%q) = %v, want %v", tt.args, got, tt.want)
		}
	}
}
