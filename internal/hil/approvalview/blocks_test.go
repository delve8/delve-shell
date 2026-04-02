package approvalview

import (
	"strings"
	"testing"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

func TestBuildSensitive(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, nil, &uivm.PendingSensitive{Command: "rm -rf /"}, nil)
	if !ok {
		t.Fatal("expected sensitive block")
	}
	if len(lines) < 3 {
		t.Fatalf("expected multiple lines, got %d", len(lines))
	}
	if lines[0].Kind != LineHeader || !strings.Contains(strings.ToLower(lines[0].Text), "sensitive") {
		t.Fatalf("unexpected header line: %#v", lines[0])
	}
}

func TestBuildApprovalRiskAndSummary(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, &uivm.PendingApproval{
		Command:   "kubectl get pods",
		RiskLevel: hiltypes.RiskLevelReadOnly,
		SkillName: "k8s",
		Summary:   "list pods",
		Reason:    "debug",
	}, nil, nil)
	if !ok {
		t.Fatal("expected approval block")
	}
	var hasRisk, hasExecCmd bool
	for _, l := range lines {
		if l.Kind == LineRiskReadOnly {
			hasRisk = true
		}
		if l.Kind == LineExec && strings.Contains(l.Text, "kubectl get pods") {
			hasExecCmd = true
		}
	}
	if !hasRisk {
		t.Fatalf("expected read-only risk line, got %#v", lines)
	}
	if !hasExecCmd {
		t.Fatalf("expected command on separate exec line, got %#v", lines)
	}
}
