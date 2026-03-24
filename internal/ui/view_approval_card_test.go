package ui

import (
	"strings"
	"testing"

	"delve-shell/internal/agent"
)

func TestAppendApprovalViewportContent_sensitive(t *testing.T) {
	m := Model{
		Layout: LayoutState{Width: 80},
		Approval: ApprovalState{
			PendingSensitive: &agent.SensitiveConfirmationRequest{
				Command: "rm -rf /",
			},
		},
	}
	var b strings.Builder
	if !m.appendApprovalViewportContent(&b) {
		t.Fatal("expected true")
	}
	out := b.String()
	if !strings.Contains(out, "rm -rf /") {
		t.Fatalf("missing command in output: %q", out)
	}
}

func TestAppendApprovalViewportContent_pendingRisk(t *testing.T) {
	m := Model{
		Layout: LayoutState{Width: 80},
		Approval: ApprovalState{
			Pending: &agent.ApprovalRequest{
				Command:   "kubectl get pods",
				RiskLevel: "read_only",
				SkillName: "k8s",
				Summary:   "list pods",
				Reason:    "debug",
			},
		},
	}
	var b strings.Builder
	if !m.appendApprovalViewportContent(&b) {
		t.Fatal("expected true")
	}
	out := b.String()
	if !strings.Contains(out, "kubectl get pods") || !strings.Contains(out, "k8s") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestAppendApprovalViewportContent_none(t *testing.T) {
	var m Model
	var b strings.Builder
	if m.appendApprovalViewportContent(&b) {
		t.Fatal("expected false")
	}
	if b.Len() != 0 {
		t.Fatalf("expected empty builder, got %q", b.String())
	}
}
