package ui

import (
	"strings"
	"testing"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/ui/uivm"
)

func TestAppendPendingChoiceCardToMessages_sensitive(t *testing.T) {
	m := Model{
		layout: LayoutState{Width: 80},
		ChoiceCard: ChoiceCardState{
			pendingSensitive: &uivm.PendingSensitive{
				Command: "rm -rf /",
			},
		},
	}
	m.appendPendingChoiceCardToMessages()
	joined := strings.Join(m.messages, "\n")
	if !strings.Contains(joined, "rm -rf /") {
		t.Fatalf("missing command in transcript: %q", joined)
	}
}

func TestAppendPendingChoiceCardToMessages_pendingRisk(t *testing.T) {
	m := Model{
		layout: LayoutState{Width: 80},
		ChoiceCard: ChoiceCardState{
			pending: &uivm.PendingApproval{
				Command:   "kubectl get pods",
				RiskLevel: hiltypes.RiskLevelReadOnly,
				SkillName: "k8s",
				Summary:   "list pods",
				Reason:    "debug",
			},
		},
	}
	m.appendPendingChoiceCardToMessages()
	joined := strings.Join(m.messages, "\n")
	if !strings.Contains(joined, "kubectl get pods") || !strings.Contains(joined, "k8s") {
		t.Fatalf("unexpected transcript: %q", joined)
	}
}

func TestAppendPendingChoiceCardToMessages_pendingMultilineCommandPreserved(t *testing.T) {
	m := Model{
		layout: LayoutState{Width: 120},
		ChoiceCard: ChoiceCardState{
			pending: &uivm.PendingApproval{
				Command:   "kubectl get nodes \\\n  -o wide\nkubectl get pods -A",
				RiskLevel: hiltypes.RiskLevelReadOnly,
			},
		},
	}
	m.appendPendingChoiceCardToMessages()
	joined := strings.Join(m.messages, "\n")
	if !strings.Contains(joined, "kubectl get nodes \\") {
		t.Fatalf("missing first command line: %q", joined)
	}
	if !strings.Contains(joined, "\n  -o wide") {
		t.Fatalf("missing preserved continuation line: %q", joined)
	}
	if !strings.Contains(joined, "\nkubectl get pods -A") {
		t.Fatalf("missing preserved third line: %q", joined)
	}
}

func TestAppendPendingChoiceCardToMessages_none(t *testing.T) {
	var m Model
	m.appendPendingChoiceCardToMessages()
	if len(m.messages) != 0 {
		t.Fatalf("expected no messages, got %d", len(m.messages))
	}
}
