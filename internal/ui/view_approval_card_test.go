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

func TestAppendPendingChoiceCardToMessages_none(t *testing.T) {
	var m Model
	m.appendPendingChoiceCardToMessages()
	if len(m.messages) != 0 {
		t.Fatalf("expected no messages, got %d", len(m.messages))
	}
}
