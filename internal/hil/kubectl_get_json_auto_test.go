package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestDebugKubectlGetPodsFieldSelectorJSON(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())
	cmd := "kubectl get pods -A --field-selector=status.phase=Running -o json"
	if !w.CommandAllowsAutoApprove(cmd) {
		t.Fatal("expected auto-approve")
	}
	pol := config.KubectlReadOnlyCLIPolicyForTest()
	args := []string{"kubectl", "get", "pods", "-A", "--field-selector=status.phase=Running", "-o", "json"}
	if !MatchReadOnlyCLIArgv(args, &pol) {
		t.Fatal("expected argv match")
	}
}
